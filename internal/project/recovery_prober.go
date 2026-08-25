// SPDX-License-Identifier: AGPL-3.0-or-later

package project

import (
	"context"
	"fmt"
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

// RecoveryProber is a background goroutine that, whenever at least one agent
// is operating in failover (primary_provider set), periodically re-probes
// each such primary provider's reachability. After recoveryRecoveredThreshold
// consecutive healthy probes it broadcasts provider.primary_recovered and
// records a feed notification so an operator can restore the agent.
//
// It stays idle — no network traffic at all — whenever no agent is in
// failover, and re-checks that condition every probe interval.
type RecoveryProber struct {
	p          *Project
	httpClient *http.Client

	mu                 sync.Mutex
	consecutiveSuccess map[string]int // provider name -> consecutive healthy probes
}

// newRecoveryProber creates a RecoveryProber bound to p. Not started until Start is called.
func newRecoveryProber(p *Project) *RecoveryProber {
	return &RecoveryProber{
		p:                  p,
		consecutiveSuccess: make(map[string]int),
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

// probe scans the current agent roster for primary providers under
// failover, probes each exactly once, and broadcasts recovery once a
// provider crosses recoveryRecoveredThreshold consecutive healthy probes.
func (rp *RecoveryProber) probe(ctx context.Context) {
	cfg := rp.p.Config()

	names := make(map[string]bool)
	for _, ag := range cfg.Agents {
		if ag.PrimaryProvider != "" {
			names[ag.PrimaryProvider] = true
		}
	}

	rp.mu.Lock()
	for tracked := range rp.consecutiveSuccess {
		if !names[tracked] {
			delete(rp.consecutiveSuccess, tracked)
		}
	}
	rp.mu.Unlock()

	if len(names) == 0 {
		return // idle — no agent in failover, no network traffic this cycle
	}

	for name := range names {
		prov, ok := rp.findProviderConfig(name)
		healthy := ok && agent.ProbeProviderHealth(ctx, rp.httpClient, prov, recoveryProbeTimeout)

		rp.mu.Lock()
		if healthy {
			rp.consecutiveSuccess[name]++
		} else {
			rp.consecutiveSuccess[name] = 0
		}
		count := rp.consecutiveSuccess[name]
		rp.mu.Unlock()

		if count == recoveryRecoveredThreshold {
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
