package task

import (
	"encoding/base64"
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
		content  []byte
		field    string // human-readable name for error messages
	}

	pkcs12Raw := payload.PKCS12Base64
	if strings.TrimSpace(pkcs12Raw) == "" {
		pkcs12Raw = payload.PKCS12
	}

	var pkcs12Bytes []byte
	if strings.TrimSpace(pkcs12Raw) != "" {
		decoded, err := base64.StdEncoding.DecodeString(pkcs12Raw)
		if err != nil {
			return fmt.Errorf("cert_save: decode pkcs12 payload: %w", err)
		}
		pkcs12Bytes = decoded
	}

	entries := []fileEntry{
		{action.CertFile, []byte(payload.Cert), "cert"},
		{action.ChainFile, []byte(payload.Chain), "chain"},
		{action.FullchainFile, []byte(payload.Fullchain), "fullchain"},
		{action.KeyFile, []byte(payload.PrivateKey), "private_key"},
		{action.PKCS12File, pkcs12Bytes, "pkcs12"},
	}

	for _, e := range entries {
		if strings.TrimSpace(e.filename) == "" {
			continue
		}
		if len(e.content) == 0 {
			return fmt.Errorf("cert_save: payload contains no %s", e.field)
		}
		dest := filepath.Join(dir, e.filename)
		if err := os.WriteFile(dest, e.content, 0600); err != nil {
			return fmt.Errorf("cert_save: write %s to %q: %w", e.field, dest, err)
		}
	}

	// partContent maps part names to their PEM content for combined file assembly.
	partContent := map[string]string{
		"cert":      payload.Cert,
		"chain":     payload.Chain,
		"fullchain": payload.Fullchain,
		"key":       payload.PrivateKey,
	}

	for _, cf := range action.CombinedFiles {
		var buf strings.Builder
		for _, part := range cf.Parts {
			piece := partContent[part]
			if strings.TrimSpace(piece) == "" {
				return fmt.Errorf("cert_save: combined file %q: payload contains no %s", cf.Filename, part)
			}
			// Ensure each PEM block ends with exactly one newline before the next.
			buf.WriteString(strings.TrimRight(piece, "\n"))
			buf.WriteByte('\n')
		}
		dest := filepath.Join(dir, cf.Filename)
		if err := os.WriteFile(dest, []byte(buf.String()), 0600); err != nil {
			return fmt.Errorf("cert_save: write combined file %q: %w", cf.Filename, err)
		}
	}

	return nil
}
