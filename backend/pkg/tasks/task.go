// Package tasks executes user-defined HTTP webhook tasks that fire when a
// record's IP address or certificate is renewed.
package tasks

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/logging/log"
	"github.com/valentin-kaiser/hdns/pkg/database"
	"github.com/valentin-kaiser/hdns/pkg/database/schema"
	"software.sslmate.com/src/go-pkcs12"
)

// Trigger identifies the kind of renewal event that fires a task.
type Trigger int8

const (
	// TriggerIP fires tasks when a record's A record (IP) is updated.
	TriggerIP Trigger = 1
	// TriggerCert fires tasks when a record's certificate is renewed.
	TriggerCert Trigger = 2
	// TriggerBoth marks a task that fires on both events.
	TriggerBoth Trigger = 3
)

// requestTimeout bounds how long a single webhook invocation may take.
const requestTimeout = 30 * time.Second

const (
	CertificateFormatPEM    = "pem"
	CertificateFormatPKCS12 = "pkcs12"
)

var client = &http.Client{Timeout: requestTimeout}

// matchesTrigger reports whether a task configured with taskTrigger should run
// for the given event.
func matchesTrigger(taskTrigger int8, event Trigger) bool {
	return taskTrigger == int8(event) || taskTrigger == int8(TriggerBoth)
}

// Execute runs a single webhook task once without persisting its result and
// returns a short status string (e.g. the HTTP status code). It is used to
// test a task configuration on demand.
func Execute(ctx context.Context, task *schema.Task) (string, error) {
	return executeTask(ctx, task)
}

// FireTasks runs all enabled tasks attached to the given record that match the
// provided event. Failures are logged and recorded on the task but never
// abort the caller.
func FireTasks(ctx context.Context, recordID int64, event Trigger) {
	var tasks []*schema.Task
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var qerr error
		tasks, qerr = q.ListEnabledTasksByRecord(ctx, recordID)
		return qerr
	})
	if err != nil {
		log.Error().Err(err).Msgf("[TASK] failed to load tasks for record %d", recordID)
		return
	}

	for _, task := range tasks {
		if !matchesTrigger(task.TriggerOn, event) {
			continue
		}
		runTask(ctx, task)
	}
}

// runTask executes a single webhook task and persists its result.
func runTask(ctx context.Context, task *schema.Task) {
	status, runErr := executeTask(ctx, task)

	result := schema.UpdateTaskResultParams{
		ID:         task.ID,
		LastRun:    sql.NullTime{Time: time.Now(), Valid: true},
		LastStatus: sql.NullString{String: status, Valid: status != ""},
	}
	if runErr != nil {
		result.LastError = sql.NullString{String: runErr.Error(), Valid: true}
		log.Error().Err(runErr).Msgf("[TASK] task %q (record %d) failed", task.Name, task.RecordID)
	} else {
		log.Info().Msgf("[TASK] task %q (record %d) completed with status %s", task.Name, task.RecordID, status)
	}

	derr := database.HDNS().Query(func(q *schema.Queries) error {
		return q.UpdateTaskResult(ctx, result)
	})
	if derr != nil {
		log.Error().Err(derr).Msgf("[TASK] failed to persist result for task %q", task.Name)
	}
}

// executeTask performs the HTTP webhook call described by the task and returns
// a short status string (e.g. the HTTP status code) on success.
func executeTask(ctx context.Context, task *schema.Task) (string, error) {
	endpoint := strings.TrimSpace(task.Url)
	if endpoint == "" {
		return "", apperror.NewError("task url is empty")
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", apperror.NewError("invalid task url").AddError(err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", apperror.NewError("task url must use http or https")
	}

	method := strings.ToUpper(strings.TrimSpace(task.Method))
	if method == "" {
		method = http.MethodPost
	}

	bodyStr, err := renderBody(ctx, task)
	if err != nil {
		return "", apperror.Wrap(err)
	}

	var body io.Reader
	if bodyStr != "" {
		body = bytes.NewBufferString(bodyStr)
	}

	rctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(rctx, method, endpoint, body)
	if err != nil {
		return "", apperror.NewError("failed to build task request").AddError(err)
	}

	headers, err := DecodeHeaders(task.Headers)
	if err != nil {
		return "", apperror.Wrap(err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Content-Type") == "" && body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", apperror.NewError("task request failed").AddError(err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))

	status := fmt.Sprintf("%d", resp.StatusCode)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return status, apperror.NewErrorf("task endpoint returned HTTP %d", resp.StatusCode)
	}
	return status, nil
}

// renderBody builds the request body for a task. When the task is configured to
// include the certificate, all four certificate files are injected into the body.
// Bodies may reference the material via the following placeholders:
//
//	{{cert}}             – leaf certificate only (cert.pem)
//	{{chain}}            – intermediate certificate(s) only (chain.pem)
//	{{fullchain}}        – leaf + intermediates (fullchain.pem)
//	{{private_key}}      – private key (privkey.pem)
//	{{certificate}}      – alias for {{fullchain}} (backward compatibility)
//	{{pkcs12}}           – PKCS#12 archive (base64-encoded)
//	{{pkcs12_base64}}    – alias for {{pkcs12}}
//	{{certificate_format}} – selected output format (pem|pkcs12)
//
// The _json variants of each placeholder are JSON-string escaped for safe
// embedding in JSON bodies. When the body is empty, a default JSON payload
// containing all four fields is sent.
func renderBody(ctx context.Context, task *schema.Task) (string, error) {
	body := ""
	if task.Body.Valid {
		body = task.Body.String
	}

	if !task.IncludeCertificate {
		return body, nil
	}

	m, err := loadCertificateMaterial(ctx, task.RecordID)
	if err != nil {
		return "", apperror.Wrap(err)
	}

	format := ResolveCertificateFormat(task.CertificateFormat, body)
	pkcs12Base64 := ""
	if format == CertificateFormatPKCS12 {
		pkcs12Base64, err = buildPKCS12Base64(m)
		if err != nil {
			return "", apperror.Wrap(err)
		}
	}

	if strings.TrimSpace(body) == "" {
		payload := map[string]string{
			"cert":               m.cert,
			"chain":              m.chain,
			"fullchain":          m.fullchain,
			"private_key":        m.key,
			"certificate":        m.fullchain,
			"certificate_format": format,
		}
		if pkcs12Base64 != "" {
			payload["pkcs12"] = pkcs12Base64
			payload["pkcs12_base64"] = pkcs12Base64
		}

		payloadRaw, merr := json.Marshal(payload)
		if merr != nil {
			return "", apperror.NewError("failed to encode certificate payload").AddError(merr)
		}
		return string(payloadRaw), nil
	}

	replacer := strings.NewReplacer(
		"{{cert}}", m.cert,
		"{{chain}}", m.chain,
		"{{fullchain}}", m.fullchain,
		"{{certificate}}", m.fullchain,
		"{{private_key}}", m.key,
		"{{pkcs12}}", pkcs12Base64,
		"{{pkcs12_base64}}", pkcs12Base64,
		"{{certificate_format}}", format,
		"{{cert_json}}", jsonEscape(m.cert),
		"{{chain_json}}", jsonEscape(m.chain),
		"{{fullchain_json}}", jsonEscape(m.fullchain),
		"{{certificate_json}}", jsonEscape(m.fullchain),
		"{{private_key_json}}", jsonEscape(m.key),
		"{{pkcs12_json}}", jsonEscape(pkcs12Base64),
		"{{pkcs12_base64_json}}", jsonEscape(pkcs12Base64),
		"{{certificate_format_json}}", jsonEscape(format),
	)
	return replacer.Replace(body), nil
}

// ResolveCertificateFormat normalizes the configured format and falls back to
// placeholder-based inference only when no explicit format is configured.
func ResolveCertificateFormat(format, body string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case CertificateFormatPEM:
		return CertificateFormatPEM
	case CertificateFormatPKCS12:
		return CertificateFormatPKCS12
	}
	if requiresPKCS12(body) {
		return CertificateFormatPKCS12
	}
	return CertificateFormatPEM
}

func requiresPKCS12(body string) bool {
	return strings.Contains(body, "{{pkcs12}}") || strings.Contains(body, "{{pkcs12_base64}}")
}

// certMaterial holds the four PEM files produced by a certificate issuance.
type certMaterial struct {
	cert      string // cert.pem      – leaf certificate only
	chain     string // chain.pem     – intermediate certificate(s) only
	fullchain string // fullchain.pem – cert + chain
	key       string // privkey.pem   – private key
}

// loadCertificateMaterial reads all four issued certificate files for the
// given record from disk. CertPath points to fullchain.pem and KeyPath to
// privkey.pem; cert.pem and chain.pem are derived from the same directory.
func loadCertificateMaterial(ctx context.Context, recordID int64) (certMaterial, error) {
	var cert *schema.Certificate
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var qerr error
		cert, qerr = q.GetCertificateByRecord(ctx, recordID)
		return qerr
	})
	if err != nil {
		return certMaterial{}, apperror.NewError("no certificate available for this record").AddError(err)
	}
	if cert.CertPath == "" || cert.KeyPath == "" {
		return certMaterial{}, apperror.NewError("certificate has not been issued yet")
	}

	dir := filepath.Dir(cert.CertPath)

	readFile := func(name string) (string, error) {
		b, rerr := os.ReadFile(filepath.Join(dir, name))
		if rerr != nil {
			return "", apperror.NewErrorf("failed to read %s", name).AddError(rerr)
		}
		return string(b), nil
	}

	var m certMaterial
	if m.cert, err = readFile("cert.pem"); err != nil {
		return certMaterial{}, err
	}
	if m.chain, err = readFile("chain.pem"); err != nil {
		return certMaterial{}, err
	}
	if m.fullchain, err = readFile("fullchain.pem"); err != nil {
		return certMaterial{}, err
	}
	if m.key, err = readFile("privkey.pem"); err != nil {
		return certMaterial{}, err
	}

	return m, nil
}

// jsonEscape returns the JSON string encoding of s without the surrounding
// quotes, suitable for embedding inside a larger JSON document.
func jsonEscape(s string) string {
	b, err := json.Marshal(s)
	if err != nil || len(b) < 2 {
		return s
	}
	return string(b[1 : len(b)-1])
}

func buildPKCS12Base64(m certMaterial) (string, error) {
	leaf, err := parseFirstCertificatePEM(m.cert)
	if err != nil {
		return "", err
	}

	intermediates, err := parseCertificateChainPEM(m.chain)
	if err != nil {
		return "", err
	}

	key, err := parsePrivateKeyPEM(m.key)
	if err != nil {
		return "", err
	}

	pfx, err := pkcs12.Encode(rand.Reader, key, leaf, intermediates, "")
	if err != nil {
		return "", apperror.NewError("failed to encode pkcs12 archive").AddError(err)
	}

	return base64.StdEncoding.EncodeToString(pfx), nil
}

func parseFirstCertificatePEM(raw string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, apperror.NewError("failed to decode certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, apperror.NewError("failed to parse certificate PEM").AddError(err)
	}
	return cert, nil
}

func parseCertificateChainPEM(raw string) ([]*x509.Certificate, error) {
	chain := make([]*x509.Certificate, 0)
	data := []byte(raw)
	for len(data) > 0 {
		block, rest := pem.Decode(data)
		if block == nil {
			break
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, apperror.NewError("failed to parse chain certificate PEM").AddError(err)
		}
		chain = append(chain, cert)
		data = rest
	}
	return chain, nil
}

func parsePrivateKeyPEM(raw string) (any, error) {
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, apperror.NewError("failed to decode private key PEM")
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

	return nil, apperror.NewError("failed to parse private key PEM")
}
