// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"strings"
	"testing"
)

// TestFormatOllamaSummary_AllFields verifies the summary line includes every
// expected segment when a full `done:true` payload is present, using the
// example payload the user provided.
func TestFormatOllamaSummary_AllFields(t *testing.T) {
	done := map[string]any{
		"done":                 true,
		"done_reason":          "stop",
		"total_duration":       float64(89790564209),
		"load_duration":        float64(15997894417),
		"prompt_eval_count":    float64(265),
		"prompt_eval_duration": float64(548814458),
		"eval_count":           float64(2480),
		"eval_duration":        float64(63366898226),
	}

	got := formatOllamaSummary(done)

	wantContains := []string{
		"# summary:",
		"done_reason=stop",
		"total=1m29.791s",
		"load=15.998s",
		"prompt_eval=265",
		"tok/s",
		"eval=2480",
	}
	for _, w := range wantContains {
		if !strings.Contains(got, w) {
			t.Errorf("summary missing %q\nfull line: %s", w, got)
		}
	}
}

// TestFormatOllamaSummary_PartialFields verifies graceful degradation when
// only some stats are present (e.g. older Ollama versions or generate vs chat).
func TestFormatOllamaSummary_PartialFields(t *testing.T) {
	done := map[string]any{
		"done":           true,
		"total_duration": float64(1_500_000_000), // 1.5 s
		"eval_count":     float64(42),
	}

	got := formatOllamaSummary(done)
	if !strings.Contains(got, "total=1.5s") {
		t.Errorf("expected total=1.5s in %q", got)
	}
	if !strings.Contains(got, "eval=42") {
		t.Errorf("expected eval=42 in %q", got)
	}
	if strings.Contains(got, "load=") {
		t.Errorf("unexpected load= segment when load_duration is absent: %q", got)
	}
}
