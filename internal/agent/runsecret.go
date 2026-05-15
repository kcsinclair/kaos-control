// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateRunSecret returns a cryptographically random 64-character hex string
// suitable for use as a per-run HMAC or bearer secret (FR5, NFR3).
func GenerateRunSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating run secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}
