// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

// RunError wraps a driver failure with a classification the queue dispatcher
// can act on. Only transient, potentially-recoverable failures are wrapped
// (see classifyHTTPFailure); everything else is returned as a plain error so
// it surfaces as a hard failure, unchanged from prior behaviour.
type RunError struct {
	Err  error
	Kind RateLimitKind
}

func (e *RunError) Error() string { return e.Err.Error() }
func (e *RunError) Unwrap() error { return e.Err }

// classifyHTTPFailure classifies an openai-compatible driver HTTP failure
// into a RateLimitKind so the queue dispatcher can decide between an
// automated provider failover and its standard rate-limit pause:
//
//   - statusCode == 0 (client.Do returned an error: connection refused, DNS
//     failure, TLS handshake failure, timeout) → unreachable.
//   - statusCode == 529 (Anthropic-style overload) → overloaded.
//   - statusCode == 429 (rate limit / quota) → rate_limit.
//   - statusCode is a gateway error (502/503/504) → unreachable — the
//     upstream itself is down or unroutable, not merely throttling.
//   - anything else (4xx auth/validation errors, malformed requests, etc.)
//     is not classified: ok is false, and the caller should treat it as a
//     hard failure.
func classifyHTTPFailure(statusCode int) (kind RateLimitKind, ok bool) {
	switch {
	case statusCode == 0:
		return RateLimitKindUnreachable, true
	case statusCode == 529:
		return RateLimitKindOverloaded, true
	case statusCode == 429:
		return RateLimitKindRateLimit, true
	case statusCode == 502 || statusCode == 503 || statusCode == 504:
		return RateLimitKindUnreachable, true
	default:
		return "", false
	}
}

// wrapHTTPError classifies err (from an openai-compatible driver HTTP call)
// by statusCode and, when classifiable, returns it wrapped in a *RunError so
// downstream failover logic can react to it. Unclassifiable errors (e.g. a
// plain 400/401/404) are returned unchanged.
func wrapHTTPError(err error, statusCode int) error {
	kind, ok := classifyHTTPFailure(statusCode)
	if !ok {
		return err
	}
	return &RunError{Err: err, Kind: kind}
}
