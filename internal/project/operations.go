// SPDX-License-Identifier: AGPL-3.0-or-later

package project

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// maxHistoryEntries bounds the failover history log so operations.yaml
// cannot grow without limit over the life of a long-running project. Oldest
// entries are dropped first.
const maxHistoryEntries = 500

// ProviderModel names a provider+model pair. Provider/model names only —
// never secret material (NFR-1).
type ProviderModel struct {
	Provider string `yaml:"provider,omitempty"`
	Model    string `yaml:"model,omitempty"`
}

// AgentOperationalState is one agent's live provider state: its declared
// primary and its currently-active provider/model, plus the context of the
// most recent switch away from primary (if any).
type AgentOperationalState struct {
	Agent   string        `yaml:"agent"`
	Primary ProviderModel `yaml:"primary,omitempty"`
	Active  ProviderModel `yaml:"active,omitempty"`

	// SwitchedAt is the Unix time of the most recent switch away from
	// Primary. Zero when the agent is on its primary provider.
	SwitchedAt int64  `yaml:"switched_at,omitempty"`
	Reason     string `yaml:"reason,omitempty"`

	// ResetsAtUnix + Bucket carry rate-limit context (FR-3.3) so the UI can
	// show when the primary is expected to become usable again.
	ResetsAtUnix int64  `yaml:"resets_at_unix,omitempty"`
	Bucket       string `yaml:"bucket,omitempty"`

	// PartialPause records that this agent's jobs are paused because it has
	// no secondary to fail over to while the rest of the project proceeds
	// (FR-3.4).
	PartialPause bool `yaml:"partial_pause,omitempty"`

	// AwaitingOperatorDecision records that a job for this agent was
	// interrupted with a suspected partial commit and needs an operator
	// decision before it can be restarted or rolled back (FR-7.3).
	AwaitingOperatorDecision bool   `yaml:"awaiting_operator_decision,omitempty"`
	AwaitingDecisionJobID    string `yaml:"awaiting_decision_job_id,omitempty"`
}

// IsFailedOver reports whether the agent is currently active on a provider
// other than its declared primary.
func (s AgentOperationalState) IsFailedOver() bool {
	return s.Primary.Provider != "" && s.Active.Provider != s.Primary.Provider
}

// ProviderReachability is the most recently probed reachability of one
// provider.
type ProviderReachability struct {
	Healthy      bool  `yaml:"healthy"`
	LastProbedAt int64 `yaml:"last_probed_at,omitempty"`
	// Since is the Unix time the current Healthy value started holding —
	// reset whenever a probe flips the value.
	Since int64 `yaml:"since,omitempty"`
}

// FailoverHistoryEntry is one recorded transition (switch, restore,
// failover, pause, retry) for the observability log (FR-10.1).
type FailoverHistoryEntry struct {
	At           int64  `yaml:"at"`
	Agent        string `yaml:"agent,omitempty"`
	Action       string `yaml:"action"` // switch | restore | failover | pause_queue | retry_in_place | fail_run
	FromProvider string `yaml:"from_provider,omitempty"`
	ToProvider   string `yaml:"to_provider,omitempty"`
	Reason       string `yaml:"reason,omitempty"`
}

// Operations is the authoritative, git-ignored runtime-state store at the
// project root (operations.yaml). It holds per-agent active-vs-primary
// state, provider reachability, per-provider disconnect events, and a
// failover history log. It carries no secret material — provider/model
// names only (NFR-1). Access is safe for concurrent use.
type Operations struct {
	mu   sync.Mutex
	path string

	Agents       map[string]AgentOperationalState `yaml:"agents,omitempty"`
	Reachability map[string]ProviderReachability  `yaml:"reachability,omitempty"`
	// Disconnects maps provider name to the Unix timestamps of recorded
	// disconnect occurrences (already collapsed per the backoff-window rule
	// in RecordDisconnect), used for the rolling-hour counter (FR-6.3/6.4).
	Disconnects map[string][]int64     `yaml:"disconnects,omitempty"`
	History     []FailoverHistoryEntry `yaml:"history,omitempty"`
}

// operationsPath returns the absolute path to operations.yaml at the
// project root. Already listed in .gitignore.
func operationsPath(projectRoot string) string {
	return filepath.Join(projectRoot, "operations.yaml")
}

// LoadOperations reads operations.yaml from the project root. A missing
// file is not an error — it returns an empty, ready-to-use store bound to
// that path so the first Save creates it.
func LoadOperations(projectRoot string) (*Operations, error) {
	path := operationsPath(projectRoot)
	ops := &Operations{path: path}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ops, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, ops); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	ops.path = path
	return ops, nil
}

// Save atomically persists the store to operations.yaml: write to a temp
// file in the same directory, then os.Rename over the destination, so a
// crash mid-write leaves the previous file intact and parseable.
func (o *Operations) Save() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.saveLocked()
}

func (o *Operations) saveLocked() error {
	data, err := yaml.Marshal(o)
	if err != nil {
		return fmt.Errorf("marshalling operations state: %w", err)
	}
	dir := filepath.Dir(o.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating dir for %s: %w", o.path, err)
	}
	tmp := o.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("writing tmp operations state: %w", err)
	}
	if err := os.Rename(tmp, o.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming tmp operations state: %w", err)
	}
	return nil
}

// AgentState returns a copy of agentName's operational state, or
// ok=false when the agent has never had one recorded (i.e. it has always
// been on its primary provider).
func (o *Operations) AgentState(agentName string) (AgentOperationalState, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	s, ok := o.Agents[agentName]
	return s, ok
}

// SetAgentState records agentName's operational state and persists it.
func (o *Operations) SetAgentState(state AgentOperationalState) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.Agents == nil {
		o.Agents = make(map[string]AgentOperationalState)
	}
	o.Agents[state.Agent] = state
	return o.saveLocked()
}

// ClearAgentState removes agentName's recorded state (restoring it to
// "on primary, no override") and persists the change.
func (o *Operations) ClearAgentState(agentName string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.Agents == nil {
		return nil
	}
	if _, ok := o.Agents[agentName]; !ok {
		return nil
	}
	delete(o.Agents, agentName)
	return o.saveLocked()
}

// AllAgentStates returns a copy of every agent's recorded operational
// state, keyed by agent name.
func (o *Operations) AllAgentStates() map[string]AgentOperationalState {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make(map[string]AgentOperationalState, len(o.Agents))
	for k, v := range o.Agents {
		out[k] = v
	}
	return out
}

// Reachability returns the last-probed reachability of the named provider,
// or ok=false when it has never been probed.
func (o *Operations) GetReachability(provider string) (ProviderReachability, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	r, ok := o.Reachability[provider]
	return r, ok
}

// AllReachability returns a copy of the reachability map for every probed
// provider.
func (o *Operations) AllReachability() map[string]ProviderReachability {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make(map[string]ProviderReachability, len(o.Reachability))
	for k, v := range o.Reachability {
		out[k] = v
	}
	return out
}

// SetReachability records the most recent probe outcome for provider and
// persists it. Since is only reset (to now) when the healthy value flips
// from the previously recorded value; a repeated probe result carries the
// prior Since forward, so the field means "since this state began" — not
// "since last probed".
func (o *Operations) SetReachability(provider string, healthy bool, now time.Time) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.Reachability == nil {
		o.Reachability = make(map[string]ProviderReachability)
	}
	prev, existed := o.Reachability[provider]
	since := now.Unix()
	if existed && prev.Healthy == healthy {
		since = prev.Since
	}
	o.Reachability[provider] = ProviderReachability{
		Healthy:      healthy,
		LastProbedAt: now.Unix(),
		Since:        since,
	}
	return o.saveLocked()
}

// RecordDisconnect records a disconnect occurrence for provider at time at,
// collapsing it into the previous occurrence when it falls inside
// collapseWindow of the last recorded occurrence — per Resolved Question 1,
// disconnects inside the active backoff window collapse to a single
// occurrence so one incident cannot trip the rolling-hour threshold
// instantly (FR-6.5). Returns true when a new (non-collapsed) occurrence
// was recorded.
func (o *Operations) RecordDisconnect(provider string, at time.Time, collapseWindow time.Duration) (bool, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.Disconnects == nil {
		o.Disconnects = make(map[string][]int64)
	}
	events := o.Disconnects[provider]
	if n := len(events); n > 0 {
		last := time.Unix(events[n-1], 0)
		if at.Sub(last) < collapseWindow {
			return false, nil // collapsed into the prior occurrence
		}
	}
	o.Disconnects[provider] = append(events, at.Unix())
	return true, o.saveLocked()
}

// DisconnectCountSince returns the number of recorded disconnect
// occurrences for provider at or after since (used for the >3/hour
// threshold, FR-6.3).
func (o *Operations) DisconnectCountSince(provider string, since time.Time) int {
	o.mu.Lock()
	defer o.mu.Unlock()
	cutoff := since.Unix()
	count := 0
	for _, at := range o.Disconnects[provider] {
		if at >= cutoff {
			count++
		}
	}
	return count
}

// AppendHistory records one transition to the failover history log and
// persists it. The log is capped at maxHistoryEntries, dropping the oldest
// entries first.
func (o *Operations) AppendHistory(entry FailoverHistoryEntry) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.History = append(o.History, entry)
	if len(o.History) > maxHistoryEntries {
		o.History = o.History[len(o.History)-maxHistoryEntries:]
	}
	return o.saveLocked()
}

// HistorySnapshot returns a copy of the full failover history log.
func (o *Operations) HistorySnapshot() []FailoverHistoryEntry {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]FailoverHistoryEntry, len(o.History))
	copy(out, o.History)
	return out
}
