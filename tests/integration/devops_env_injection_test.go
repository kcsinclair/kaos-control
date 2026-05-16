// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Verifies that pipeline steps see KC_PROJECT, KC_PORT, and KC_API_TOKEN in
// their environment, and that the injected token is a valid bearer credential
// for the kaos-control API.

import (
	"strings"
	"testing"
)

const pipelineEnvProbe = `name: Env Probe
type: build
steps:
  - name: Probe
    command: |
      printf 'PROJECT=%s\n' "$KC_PROJECT"
      printf 'PORT=%s\n' "$KC_PORT"
      printf 'TOKEN_LEN=%s\n' "${#KC_API_TOKEN}"
      curl -s -o /dev/null -w 'HTTP=%{http_code}\n' \
        -H "Authorization: Bearer $KC_API_TOKEN" \
        "http://127.0.0.1:$KC_PORT/api/p/$KC_PROJECT/artifacts"
`

func TestDevopsRun_InjectsEnvAndBearerToken(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"env-probe.yaml": pipelineEnvProbe,
	})
	env.login("admin@test.local", "admin-pass-123")

	lines, _ := runAndFetchNDJSON(t, env, "env-probe")

	var stepOutput []string
	for _, obj := range lines {
		if typ, _ := obj["type"].(string); typ == "pipeline.step.output" {
			if text, _ := obj["text"].(string); text != "" {
				stepOutput = append(stepOutput, text)
			}
		}
	}
	joined := strings.Join(stepOutput, "\n")

	cases := []struct {
		want string
		desc string
	}{
		{"PROJECT=testproject", "KC_PROJECT injected with project name"},
		{"PORT=", "KC_PORT injected (any value)"},
		{"TOKEN_LEN=64", "KC_API_TOKEN injected (64 hex chars)"},
		{"HTTP=200", "bearer token authenticates a real API call"},
	}
	for _, c := range cases {
		if !strings.Contains(joined, c.want) {
			t.Errorf("%s: expected substring %q in step output, got:\n%s", c.desc, c.want, joined)
		}
	}
}

