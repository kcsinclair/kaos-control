// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestClassifyRunError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		stderr     string
		wantReason string
	}{
		{"nil err and empty stderr", nil, "", ""},
		{"model not found sentinel", fmt.Errorf("wrap: %w", ErrModelNotFound), "", FailureReasonModelNotFound},
		{"endpoint unreachable sentinel", fmt.Errorf("wrap: %w", ErrEndpointUnreachable), "", FailureReasonEndpointUnreachable},
		{"tools unsupported sentinel", ErrToolsUnsupported, "", FailureReasonToolsUnsupported},
		{"tools silently dropped sentinel", ErrToolsSilentlyDropped, "", FailureReasonToolsUnsupported},
		{"model load timeout sentinel", ErrModelLoadTimeout, "", FailureReasonModelUnloaded},
		{"max iterations sentinel", ErrMaxIterationsReached, "", FailureReasonMaxIterationsReached},
		{"context deadline exceeded", context.DeadlineExceeded, "", FailureReasonTimeout},
		{"RunError wrapping HTTP 503", &RunError{Err: fmt.Errorf(`provider "local" returned HTTP 503: model still loading`), Kind: RateLimitKindUnreachable}, "", FailureReasonModelUnloaded},
		{"RunError unreachable kind", &RunError{Err: fmt.Errorf("dial tcp: connection refused"), Kind: RateLimitKindUnreachable}, "", FailureReasonEndpointUnreachable},
		{"401 in stderr", nil, "provider returned HTTP 401 unauthorized", FailureReasonAuthError},
		{"context length exceeded text", nil, "error: this model's maximum context length is 4096 tokens", FailureReasonContextWindowExceeded},
		{"turn token ceiling text", nil, `"finish_reason":"length"`, FailureReasonTurnTokenCeiling},
		{"connection refused text", nil, "dial tcp 127.0.0.1:8080: connect: connection refused", FailureReasonEndpointUnreachable},
		{"unclassifiable text", fmt.Errorf("something completely unrelated went wrong"), "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reason, remediation, details := ClassifyRunError(tc.err, tc.stderr, nil)
			if reason != tc.wantReason {
				t.Errorf("reason = %q, want %q", reason, tc.wantReason)
			}
			if tc.wantReason == "" {
				if remediation != nil || details != nil {
					t.Errorf("expected nil remediation/details for unclassified error, got %v / %v", remediation, details)
				}
				return
			}
			if len(remediation) == 0 {
				t.Error("expected non-empty remediation for a classified reason")
			}
			if details == nil {
				t.Error("expected non-nil errorDetails for a classified reason")
			}
		})
	}
}

func TestClassifyRunError_MasksSecretsInDetails(t *testing.T) {
	err := fmt.Errorf("%w: provider rejected request with Authorization: Bearer sk-super-secret-token-12345", ErrEndpointUnreachable)
	stderr := `{"api_key":"sk-another-secret-value"}`

	reason, _, details := ClassifyRunError(err, stderr, nil)
	if reason != FailureReasonEndpointUnreachable {
		t.Fatalf("expected endpoint_unreachable, got %q", reason)
	}

	msg, _ := details["message"].(string)
	if strings.Contains(msg, "sk-super-secret-token-12345") {
		t.Errorf("expected bearer token to be masked in message, got: %q", msg)
	}
	if !strings.Contains(msg, "***") {
		t.Errorf("expected masked message to contain a redaction marker, got: %q", msg)
	}

	excerpt, _ := details["stderr_excerpt"].(string)
	if strings.Contains(excerpt, "sk-another-secret-value") {
		t.Errorf("expected api_key to be masked in stderr_excerpt, got: %q", excerpt)
	}
}

func TestMaskSecretsInText(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`Authorization: Bearer sk-abc123.def-456`, `Authorization: Bearer ***`},
		{`"api_key": "sk-abc123"`, `"api_key": "***"`},
		{`api-key=sk-abc123`, `api-key=***`},
		{"no secrets here", "no secrets here"},
	}
	for _, tc := range cases {
		got := maskSecretsInText(tc.in)
		if got != tc.want {
			t.Errorf("maskSecretsInText(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestClassifyReason_MidStreamConnectionReset pins the classification of a
// connection dropped mid-generation. Run 8f15fc7f0fe9afa9 died with exactly
// this stderr and recorded an EMPTY failure_reason, because the classifier
// looked for "dial tcp"/"connection refused"/"eof" and this string matches none
// of them. It must NOT be reported as endpoint_unreachable: that remediation
// says "verify base_url is correct and the server is running", which is
// misleading when the server is up and merely reset an in-flight stream (here,
// a llama-swap model swap).
func TestClassifyReason_MidStreamConnectionReset(t *testing.T) {
	stderr := "read tcp 192.168.1.9:63826->192.168.1.2:7442: read: connection reset by peer"
	reason, remediation := classifyReason(nil, stderr)

	if reason != FailureReasonProviderDisconnected {
		t.Errorf("reason = %q, want %q", reason, FailureReasonProviderDisconnected)
	}
	if len(remediation) == 0 {
		t.Error("remediation is empty — the operator gets no guidance")
	}

	// A genuinely unreachable endpoint must still classify as such.
	if r, _ := classifyReason(nil, "dial tcp 192.168.1.2:7442: connect: connection refused"); r != FailureReasonEndpointUnreachable {
		t.Errorf("connection refused = %q, want %q", r, FailureReasonEndpointUnreachable)
	}

	// "unexpected EOF" is a mid-stream drop, not an unreachable endpoint — it
	// must not be swallowed by the generic branch's "eof" needle.
	if r, _ := classifyReason(nil, "stream error: unexpected EOF"); r != FailureReasonProviderDisconnected {
		t.Errorf("unexpected EOF = %q, want %q", r, FailureReasonProviderDisconnected)
	}
}
