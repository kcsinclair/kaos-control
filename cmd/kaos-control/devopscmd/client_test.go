// SPDX-License-Identifier: AGPL-3.0-or-later

package devopscmd

import (
	"testing"

	"github.com/kaos-control/kaos-control/internal/config"
)

func appWith(listen, publicHost string, tls bool) *config.App {
	a := &config.App{}
	a.Server.Listen = listen
	a.Server.PublicHost = publicHost
	a.Server.TLS.Enabled = tls
	return a
}

func TestNewClient_TargetsLocalListen_NotPublicHost(t *testing.T) {
	t.Setenv("KAOS_CONTROL_SERVER", "") // ensure no override

	cases := []struct {
		name    string
		listen  string
		public  string
		tls     bool
		wantURL string
	}{
		{"public_host is ignored", "127.0.0.1:8787", "sol.packsin.com", false, "http://127.0.0.1:8787"},
		{"wildcard bind → loopback", "0.0.0.0:9000", "sol.packsin.com", false, "http://127.0.0.1:9000"},
		{"bare :port → loopback", ":8080", "", false, "http://127.0.0.1:8080"},
		{"tls → https", "127.0.0.1:8443", "sol.packsin.com", true, "https://127.0.0.1:8443"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newClient(appWith(tc.listen, tc.public, tc.tls), authMode{})
			if c.baseURL != tc.wantURL {
				t.Errorf("baseURL = %q, want %q", c.baseURL, tc.wantURL)
			}
		})
	}
}

func TestNewClient_ServerOverrideWins(t *testing.T) {
	t.Setenv("KAOS_CONTROL_SERVER", "https://remote.example:1234/")
	c := newClient(appWith("127.0.0.1:8787", "sol.packsin.com", false), authMode{})
	if c.baseURL != "https://remote.example:1234" {
		t.Errorf("baseURL = %q, want the override with trailing slash trimmed", c.baseURL)
	}
}
