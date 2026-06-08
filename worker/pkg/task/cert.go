package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/valentin-kaiser/hdns-worker/pkg/config"
)

// save writes the configured certificate files from payload into the directory
// specified by action.CertDir. Only files with a non-empty filename in the
// action config are written; the others are silently skipped.
// The directory is created if it does not exist.
// Files are written with permissions 0600 so only the owning user can read them.
func save(action config.ActionConfig, payload Payload) error {
	dir := filepath.Clean(action.CertDir)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("cert_save: create directory %q: %w", dir, err)
	}

	type fileEntry struct {
		filename string
		content  string
		field    string // human-readable name for error messages
	}

	entries := []fileEntry{
		{action.CertFile, payload.Cert, "cert"},
		{action.ChainFile, payload.Chain, "chain"},
		{action.FullchainFile, payload.Fullchain, "fullchain"},
		{action.KeyFile, payload.PrivateKey, "private_key"},
	}

	for _, e := range entries {
		if strings.TrimSpace(e.filename) == "" {
			continue
		}
		if strings.TrimSpace(e.content) == "" {
			return fmt.Errorf("cert_save: payload contains no %s", e.field)
		}
		dest := filepath.Join(dir, e.filename)
		if err := os.WriteFile(dest, []byte(e.content), 0600); err != nil {
			return fmt.Errorf("cert_save: write %s to %q: %w", e.field, dest, err)
		}
	}

	return nil
}
