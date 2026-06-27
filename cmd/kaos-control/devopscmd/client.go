// SPDX-License-Identifier: AGPL-3.0-or-later

package devopscmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kaos-control/kaos-control/internal/config"
)

// client is a thin HTTP client that attaches identity headers to every request.
type client struct {
	baseURL    string
	identity   authMode
	httpClient *http.Client
}

// newClient builds a client from the app config and resolved identity.
// The base URL is derived from App.Server.Listen, honouring PublicHost/TLS.
func newClient(appCfg *config.App, identity authMode) *client {
	listen := appCfg.Server.Listen
	scheme := "http"
	if appCfg.Server.TLS.Enabled {
		scheme = "https"
	}

	host := appCfg.Server.PublicHost
	if host == "" {
		host = listen
		// If listen is :port (no host), prepend 127.0.0.1.
		if strings.HasPrefix(host, ":") {
			host = "127.0.0.1" + host
		}
	}

	return &client{
		baseURL:    scheme + "://" + host,
		identity:   identity,
		httpClient: &http.Client{},
	}
}

// get performs GET baseURL+path and returns the response body.
// Maps 401→exitIdentityUnresolved, 403→exitForbidden, other non-2xx→exitOpFailed.
func (c *client) get(path string) (string, int) {
	return c.do(http.MethodGet, path, nil)
}

// post performs POST baseURL+path with the given JSON body and returns the response.
func (c *client) post(path string, body map[string]any) (string, int) {
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	return c.do(http.MethodPost, path, r)
}

func (c *client) do(method, path string, body io.Reader) (string, int) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building request: %v\n", err)
		return "", exitOpFailed
	}

	// Attach identity. Token values are never echoed; only the header name is set.
	if c.identity.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+c.identity.bearer)
	} else if c.identity.localEmail != "" {
		req.Header.Set("X-Kaos-Local-User", c.identity.localEmail)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error contacting server at %s: %v\n", c.baseURL, err)
		return "", exitOpFailed
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	respBody := string(data)

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		fmt.Fprintln(os.Stderr, "unauthorized: identity not accepted by server (exit 3)")
		return "", exitIdentityUnresolved
	case resp.StatusCode == http.StatusForbidden:
		// Surface the server's "role required:" message.
		var apiErr struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		if json.Unmarshal(data, &apiErr) == nil {
			msg := apiErr.Message
			if msg == "" {
				msg = apiErr.Error
			}
			fmt.Fprintln(os.Stderr, msg)
		} else {
			fmt.Fprintln(os.Stderr, "forbidden: insufficient role")
		}
		return "", exitForbidden
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return respBody, exitOK
	default:
		var apiErr struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		if json.Unmarshal(data, &apiErr) == nil {
			msg := apiErr.Message
			if msg == "" {
				msg = apiErr.Error
			}
			fmt.Fprintln(os.Stderr, msg)
		} else {
			fmt.Fprintf(os.Stderr, "server returned %d: %s\n", resp.StatusCode, respBody)
		}
		return "", exitOpFailed
	}
}

// loadAppConfig resolves and loads the application config.
func loadAppConfig() (*config.App, int) {
	cfgPath := defaultConfigPath()
	appCfg, err := config.LoadApp(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config from %s: %v\n", cfgPath, err)
		return nil, exitOpFailed
	}
	return appCfg, exitOK
}

func defaultConfigPath() string {
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "kaos-control", "config.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kaos-control", "config.yaml")
}

// artifactRow is a minimal representation for the list table.
type artifactRow struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Lineage string `json:"lineage"`
	Title   string `json:"title"`
}

// parseArtifactList extracts the artifacts array from a list-artifacts response.
func parseArtifactList(body string) []artifactRow {
	var wrapper struct {
		Artifacts []artifactRow `json:"artifacts"`
	}
	if err := json.Unmarshal([]byte(body), &wrapper); err != nil {
		return nil
	}
	return wrapper.Artifacts
}

// extractJSONField extracts a single top-level field from a JSON object,
// returning it as a raw JSON string. Falls back to the original body on error.
func extractJSONField(body, field string) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return body
	}
	if v, ok := m[field]; ok {
		return string(v)
	}
	return body
}
