package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/valentin-kaiser/hdns-worker/pkg/config"
)

// save writes the certificate PEM and private-key PEM from payload into
// the directory specified by action.CertDir.
// The directory is created if it does not exist.
// Files are written with permissions 0600 so only the owning user can read them.
func save(action config.ActionConfig, payload Payload) error {
	if strings.TrimSpace(payload.Certificate) == "" {
		return fmt.Errorf("cert_save: payload contains no certificate")
	}
	if strings.TrimSpace(payload.PrivateKey) == "" {
		return fmt.Errorf("cert_save: payload contains no private_key")
	}

	dir := filepath.Clean(action.CertDir)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("cert_save: create directory %q: %w", dir, err)
	}

	certPath := filepath.Join(dir, action.CertFile)
	if err := os.WriteFile(certPath, []byte(payload.Certificate), 0600); err != nil {
		return fmt.Errorf("cert_save: write certificate to %q: %w", certPath, err)
	}

	keyPath := filepath.Join(dir, action.KeyFile)
	if err := os.WriteFile(keyPath, []byte(payload.PrivateKey), 0600); err != nil {
		return fmt.Errorf("cert_save: write private key to %q: %w", keyPath, err)
	}

	return nil
}
