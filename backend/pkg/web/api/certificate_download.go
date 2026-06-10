package api

import (
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/valentin-kaiser/go-core/logging/log"
	"github.com/valentin-kaiser/hdns/pkg/database"
	"github.com/valentin-kaiser/hdns/pkg/database/schema"
	"software.sslmate.com/src/go-pkcs12"
)

func CertificateDownloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	recordID, ok := parseRecordID(w, r)
	if !ok {
		return
	}

	artifact := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("artifact")))
	if artifact == "" {
		http.Error(w, "artifact is required", http.StatusBadRequest)
		return
	}

	var record *schema.Record
	var cert *schema.Certificate
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var qerr error
		record, qerr = q.GetRecord(r.Context(), recordID)
		if qerr != nil {
			return qerr
		}
		cert, qerr = q.GetCertificateByRecord(r.Context(), recordID)
		return qerr
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "certificate not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to load certificate", http.StatusInternalServerError)
		return
	}

	if cert.CertPath == "" || cert.KeyPath == "" {
		http.Error(w, "certificate has no downloadable artifacts", http.StatusNotFound)
		return
	}

	baseName := certificateFileBase(record)
	fullchainPath := cert.CertPath
	keyPath := cert.KeyPath
	certDir := filepath.Dir(fullchainPath)

	var fileName string
	var contentType string
	var payload []byte

	switch artifact {
	case "cert":
		fileName = baseName + "-cert.pem"
		contentType = "application/x-pem-file"
		payload, err = os.ReadFile(filepath.Join(certDir, "cert.pem"))
	case "chain":
		fileName = baseName + "-chain.pem"
		contentType = "application/x-pem-file"
		payload, err = os.ReadFile(filepath.Join(certDir, "chain.pem"))
	case "fullchain":
		fileName = baseName + "-fullchain.pem"
		contentType = "application/x-pem-file"
		payload, err = os.ReadFile(fullchainPath)
	case "privkey":
		fileName = baseName + "-privkey.pem"
		contentType = "application/x-pem-file"
		payload, err = os.ReadFile(keyPath)
	case "pkcs12":
		fileName = baseName + ".p12"
		contentType = "application/x-pkcs12"
		payload, err = buildPKCS12Archive(fullchainPath, keyPath)
	default:
		http.Error(w, "unsupported artifact", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "failed to read artifact", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+fileName+"\"")
	w.WriteHeader(http.StatusOK)
	if _, werr := w.Write(payload); werr != nil {
		log.Warn().Err(werr).Msg("failed to write certificate download response")
	}
}

func parseRecordID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	recordID, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("record_id")), 10, 64)
	if err != nil || recordID <= 0 {
		http.Error(w, "valid record_id is required", http.StatusBadRequest)
		return 0, false
	}
	return recordID, true
}

func buildPKCS12Archive(fullchainPath, keyPath string) ([]byte, error) {
	fullchain, err := os.ReadFile(fullchainPath)
	if err != nil {
		return nil, err
	}
	privKey, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	leaf, rest, err := parseFirstCertificateFromPEM(fullchain)
	if err != nil {
		return nil, err
	}
	intermediates, err := parseCertificateChain(rest)
	if err != nil {
		return nil, err
	}

	key, err := parsePrivateKey(privKey)
	if err != nil {
		return nil, err
	}

	return pkcs12.Modern.Encode(key, leaf, intermediates, "")
}

func parseFirstCertificateFromPEM(raw []byte) (*x509.Certificate, []byte, error) {
	block, rest := pem.Decode(raw)
	if block == nil {
		return nil, nil, os.ErrInvalid
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, err
	}
	return cert, rest, nil
}

func parseCertificateChain(raw []byte) ([]*x509.Certificate, error) {
	chain := make([]*x509.Certificate, 0)
	for len(raw) > 0 {
		block, rest := pem.Decode(raw)
		if block == nil {
			break
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		chain = append(chain, cert)
		raw = rest
	}
	return chain, nil
}

func parsePrivateKey(raw []byte) (any, error) {
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, os.ErrInvalid
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	return nil, os.ErrInvalid
}
