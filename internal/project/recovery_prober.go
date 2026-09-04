// SPDX-License-Identifier: AGPL-3.0-or-later

package project

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/kaos-control/kaos-control/internal/agent"
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/hub"
)

// recoveryProbeTimeout bounds each individual provider health check.
const recoveryProbeTimeout = 3 * time.Second

// recoveryRecoveredThreshold is the number of consecutive healthy probes
// required before a primary provider is announced as recovered.
const recoveryRecoveredThreshold = 2

// RecoveryProber is a background goroutine that periodically probes the
// reachability of every provider bound to any configured agent — in every
// mode, including single-provider (agent-switchover-and-failover FR-5.1) —
// and records the result to operations.yaml (FR-5.2). Whenever at least one
// agent is operating in failover, it additionally announces
// provider.primary_recovered for that agent's primary once
// recoveryRecoveredThreshold consecutive healthy probes are observed —
// gated on any recorded rate-limit reset time so a quota failover doesn't
// look "recovered" merely because /v1/models responds (FR-9.3).
//
// It stays idle — no network traffic at all — whenever no agent is
// configured at all, and re-checks that condition every probe interval.
type RecoveryProber struct {
	p          *Project
	httpClient *http.Client

	mu                 sync.Mutex
	consecutiveSuccess map[string]int  // provider name -> consecutive healthy probes
	announcedRecovered map[string]bool // provider name -> already announced for the current healthy streak
}

// newRecoveryProber creates a RecoveryProber bound to p. Not started until Start is called.
func newRecoveryProber(p *Project) *RecoveryProber {
	return &RecoveryProber{
		p:                  p,
		consecutiveSuccess: make(map[string]int),
		announcedRecovered: make(map[string]bool),
	}
}

// Start launches the prober goroutine. It returns a done channel that is
// closed when the goroutine exits (on ctx cancellation), so callers can wait
// for a clean shutdown.
func (rp *RecoveryProber) Start(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		rp.loop(ctx)
	}()
	return done
}

// loop re-probes at the project's configured probe_interval_seconds (live —
// re-read every iteration so a config reload takes effect without restart).
func (rp *RecoveryProber) loop(ctx context.Context) {
	for {
		interval := time.Duration(rp.p.Config().EffectiveFailoverConfig().ProbeIntervalSeconds) * time.Second
		if interval <= 0 {
			interval = 60 * time.Second
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
			rp.probe(ctx)
		}
	}
}

// probe scans the current agent roster for every provider in use (primary
// or fallback, in any mode) and probes each exactly once, writing
// reachability to operations.yaml. Providers with at least one agent
// currently failed over from them are additionally candidates for a
// provider.primary_recovered announcement once they cross
// recoveryRecoveredThreshold consecutive healthy probes and, for a
// rate-limit failover, once the recorded reset time has passed.
func (rp *RecoveryProber) probe(ctx context.Context) {
	cfg := rp.p.Config()

	names := make(map[string]bool)
	for _, ag := range cfg.Agents {
		if ag.Provider != "" {
			names[ag.Provider] = true
		}
		if ag.FallbackProvider != "" {
			names[ag.FallbackProvider] = true
		}
	}

	rp.mu.Lock()
	for tracked := range rp.consecutiveSuccess {
		if !names[tracked] {
			delete(rp.consecutiveSuccess, tracked)
			delete(rp.announcedRecovered, tracked)
		}
	}
	rp.mu.Unlock()

	if len(names) == 0 {
		return // idle — no provider-backed agents configured
	}

	// Providers with at least one agent currently failed over from them —
	// only these are candidates for a "recovered" announcement, gated on
	// the latest recorded rate-limit reset time among them (FR-9.3). A
	// non-rate-limit failover (ResetsAtUnix == 0) is not quota-gated.
	failoverGate := make(map[string]int64) // primary provider -> latest resets_at_unix
	for _, state := range rp.p.Operations().AllAgentStates() {
		if !state.IsFailedOver() {
			continue
		}
		if _, tracked := failoverGate[state.Primary.Provider]; !tracked || state.ResetsAtUnix > failoverGate[state.Primary.Provider] {
			failoverGate[state.Primary.Provider] = state.ResetsAtUnix
		}
	}

	now := time.Now()
	for name := range names {
		prov, ok := rp.findProviderConfig(name)
		healthy := ok && agent.ProbeProviderHealth(ctx, rp.httpClient, prov, recoveryProbeTimeout)

		if err := rp.p.Operations().SetReachability(name, healthy, now); err != nil {
			slog.Warn("recovery prober: recording reachability", "project", rp.p.Entry.Name, "provider", name, "err", err)
		}

		rp.mu.Lock()
		if healthy {
			rp.consecutiveSuccess[name]++
		} else {
			rp.consecutiveSuccess[name] = 0
			rp.announcedRecovered[name] = false
		}
		count := rp.consecutiveSuccess[name]
		alreadyAnnounced := rp.announcedRecovered[name]
		rp.mu.Unlock()

		resetsAt, hasFailover := failoverGate[name]
		if !hasFailover || alreadyAnnounced || count < recoveryRecoveredThreshold {
			continue
		}
		if resetsAt > 0 && now.Before(time.Unix(resetsAt, 0)) {
			// FR-9.3: quota-gated — do not announce recovery before the
			// recorded reset time, even with healthy /v1/models probes.
			continue
		}

		rp.mu.Lock()
		rp.announcedRecovered[name] = true
		rp.mu.Unlock()

		rp.p.Hub.Broadcast(hub.Event{
			Type: "provider.primary_recovered",
			Payload: map[string]any{
				"provider": name,
				"project":  rp.p.Entry.Name,
			},
		})
		rp.p.insertFeedEvent("primary_recovered", fmt.Sprintf("Primary provider %s has recovered and is ready to be restored.", name))
	}
}

// findProviderConfig looks up name in the project's app-level provider list
// (snapshotted at Open time from OpenOptions.Providers).
func (rp *RecoveryProber) findProviderConfig(name string) (config.Provider, bool) {
	for i := range rp.p.Providers {
		if rp.p.Providers[i].Name == name {
			return rp.p.Providers[i], true
		}
	}
	return config.Provider{}, false
}
