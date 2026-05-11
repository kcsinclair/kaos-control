// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"slices"
	"testing"
)

func TestAllowedWSOrigins(t *testing.T) {
	tests := []struct {
		name       string
		listen     string
		publicHost string
		wantAll    []string // all of these must appear in the result
		wantNot    []string // none of these may appear in the result
	}{
		{
			name:       "empty PublicHost always includes localhost and 127.0.0.1",
			listen:     ":8080",
			publicHost: "",
			wantAll:    []string{"localhost", "127.0.0.1"},
		},
		{
			name:       "wildcard listen address 0.0.0.0 is not added to list",
			listen:     "0.0.0.0:8080",
			publicHost: "",
			wantAll:    []string{"localhost", "127.0.0.1"},
			wantNot:    []string{"0.0.0.0"},
		},
		{
			name:       "IPv6 wildcard :: is not added to list",
			listen:     "[::]:8080",
			publicHost: "",
			wantAll:    []string{"localhost", "127.0.0.1"},
			wantNot:    []string{"::"},
		},
		{
			name:       "single public host is appended",
			listen:     ":8080",
			publicHost: "example.test",
			wantAll:    []string{"localhost", "127.0.0.1", "example.test"},
		},
		{
			name:       "comma-separated public hosts are all appended",
			listen:     ":8080",
			publicHost: "kaos.internal, kaos-control.example.com",
			wantAll:    []string{"localhost", "127.0.0.1", "kaos.internal", "kaos-control.example.com"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &Server{
				cfg: ServerConfig{
					Listen:     tc.listen,
					PublicHost: tc.publicHost,
				},
			}
			got := s.allowedWSOrigins()

			for _, want := range tc.wantAll {
				if !slices.Contains(got, want) {
					t.Errorf("allowedWSOrigins() = %v; want %q to be present", got, want)
				}
			}
			for _, notWant := range tc.wantNot {
				if slices.Contains(got, notWant) {
					t.Errorf("allowedWSOrigins() = %v; want %q to be absent", got, notWant)
				}
			}
		})
	}
}
