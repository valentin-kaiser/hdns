package tasks

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/security"
	"github.com/valentin-kaiser/hdns/pkg/config"
)

// EncryptHeaders encrypts a JSON object of HTTP headers for storage at rest.
// An empty input yields an empty (NULL) value.
func EncryptHeaders(headersJSON string) (sql.NullString, error) {
	headersJSON = strings.TrimSpace(headersJSON)
	if headersJSON == "" {
		return sql.NullString{}, nil
	}

	// Validate that the input is a JSON object of string values.
	var probe map[string]string
	if err := json.Unmarshal([]byte(headersJSON), &probe); err != nil {
		return sql.NullString{}, apperror.NewError("headers must be a JSON object of string values").AddError(err)
	}

	keyBytes := config.EncryptionKey()
	if len(keyBytes) != 32 {
		return sql.NullString{}, apperror.NewError("invalid encryption key")
	}

	var encBuf bytes.Buffer
	cipher := security.NewAesCipher().WithPassphrase(keyBytes).Encrypt(headersJSON, &encBuf)
	if cipher.Error != nil {
		return sql.NullString{}, apperror.NewError("failed to encrypt task headers").AddError(cipher.Error)
	}

	return sql.NullString{String: encBuf.String(), Valid: true}, nil
}

// DecodeHeaders decrypts the stored headers value into a map.
func DecodeHeaders(stored sql.NullString) (map[string]string, error) {
	if !stored.Valid || stored.String == "" {
		return map[string]string{}, nil
	}

	keyBytes := config.EncryptionKey()
	if len(keyBytes) != 32 {
		return nil, apperror.NewError("invalid encryption key")
	}

	var plainBuf bytes.Buffer
	cipher := security.NewAesCipher().WithPassphrase(keyBytes).Decrypt(stored.String, &plainBuf)
	if cipher.Error != nil {
		return nil, apperror.NewError("failed to decrypt task headers").AddError(cipher.Error)
	}

	headers := map[string]string{}
	if err := json.Unmarshal(plainBuf.Bytes(), &headers); err != nil {
		return nil, apperror.NewError("failed to parse task headers").AddError(err)
	}

	return headers, nil
}
