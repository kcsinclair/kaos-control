// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

// Switchover policy action verbs (agent-switchover-and-failover FR-2.2).
// Mirrored as plain strings in internal/config (EffectiveSwitchoverPolicy)
// and internal/queue (FailoverPolicy) — both packages intentionally avoid
// importing this one (queue stays decoupled from config/agent; config is
// imported *by* agent, so the reverse import would cycle) — so the verb
// text must stay in sync across all three.
const (
	SwitchoverActionFailover     = "failover"
	SwitchoverActionPauseQueue   = "pause_queue"
	SwitchoverActionRetryInPlace = "retry_in_place"
	SwitchoverActionFailRun      = "fail_run"
)

// SwitchoverReasons is the canonical, complete list of failure reasons the
// event -> action policy must cover (FR-2.1, FR-2.3): the three
// RateLimitKind transient-failure variants, auth_error, and every
// FailureReason* this package classifies via ClassifyRunError.
var SwitchoverReasons = []string{
	string(RateLimitKindRateLimit),
	string(RateLimitKindOverloaded),
	string(RateLimitKindUnreachable),
	FailureReasonAuthError,
	FailureReasonProviderDisconnected,
	FailureReasonModelNotFound,
	FailureReasonModelUnloaded,
	FailureReasonToolsUnsupported,
	FailureReasonContextWindowExceeded,
	FailureReasonTurnTokenCeiling,
	FailureReasonMaxIterationsReached,
	FailureReasonTimeout,
}

// DefaultSwitchoverAction returns the FR-2.3 default action for reason,
// given whether automated switchover is enabled project-wide:
//   - the transient, provider-account-scoped reasons (rate_limit, overloaded,
//     unreachable, auth_error) default to "failover" when automated
//     switchover is enabled, else "pause_queue";
//   - provider_disconnected defaults to "retry_in_place" (bounded, see
//     Milestone 5) regardless of automated switchover;
//   - every other (run-level setup/limit) reason defaults to "fail_run",
//     since a different provider would not help.
func DefaultSwitchoverAction(reason string, automatedSwitchoverEnabled bool) string {
	switch reason {
	case string(RateLimitKindRateLimit), string(RateLimitKindOverloaded), string(RateLimitKindUnreachable), FailureReasonAuthError:
		if automatedSwitchoverEnabled {
			return SwitchoverActionFailover
		}
		return SwitchoverActionPauseQueue
	case FailureReasonProviderDisconnected:
		return SwitchoverActionRetryInPlace
	default:
		return SwitchoverActionFailRun
	}
}
