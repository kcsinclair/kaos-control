// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"context"
	"net/http"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
)

// ProbeProviderHealth issues a fast, bounded GET against provider's
// "v1/models" endpoint (every openai-compatible provider is expected to
// expose it) to determine reachability. It returns true when the provider
// responds at all with a non-5xx status — even a 401/404 proves the upstream
// is up and routing requests, which is all a pre-switch or recovery probe
// needs to know. Connection failures, TLS errors, and timeouts return false.
//
// Shared by the queue dispatcher's pre-switch failover check (2s timeout)
// and the project package's background recovery prober (3s timeout) so both
// use identical reachability semantics.
func ProbeProviderHealth(ctx context.Context, client *http.Client, provider config.Provider, timeout time.Duration) bool {
	if client == nil {
		client = http.DefaultClient
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	url := buildEndpointURL(provider.BaseURL, "v1/models")
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	applyProviderHeaders(req, provider)

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}
