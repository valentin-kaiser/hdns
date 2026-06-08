// Package tasks executes user-defined HTTP webhook tasks that fire when a
// record's IP address or certificate is renewed.
package tasks

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/logging/log"
	"github.com/valentin-kaiser/hdns/pkg/database"
	"github.com/valentin-kaiser/hdns/pkg/database/schema"
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
// include the certificate, the issued certificate and private key are injected
// into the body. Bodies may reference the material via the {{certificate}},
// {{private_key}}, {{certificate_json}} and {{private_key_json}} placeholders;
// the _json variants are JSON-string escaped for safe embedding in JSON. When
// the body is empty, a default JSON payload containing both is sent.
func renderBody(ctx context.Context, task *schema.Task) (string, error) {
	body := ""
	if task.Body.Valid {
		body = task.Body.String
	}

	if !task.IncludeCertificate {
		return body, nil
	}

	cert, key, err := loadCertificateMaterial(ctx, task.RecordID)
	if err != nil {
		return "", apperror.Wrap(err)
	}

	if strings.TrimSpace(body) == "" {
		payload, merr := json.Marshal(map[string]string{
			"certificate": cert,
			"private_key": key,
		})
		if merr != nil {
			return "", apperror.NewError("failed to encode certificate payload").AddError(merr)
		}
		return string(payload), nil
	}

	replacer := strings.NewReplacer(
		"{{certificate}}", cert,
		"{{private_key}}", key,
		"{{certificate_json}}", jsonEscape(cert),
		"{{private_key_json}}", jsonEscape(key),
	)
	return replacer.Replace(body), nil
}

// loadCertificateMaterial reads the issued certificate chain and private key
// for the given record from disk.
func loadCertificateMaterial(ctx context.Context, recordID int64) (string, string, error) {
	var cert *schema.Certificate
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var qerr error
		cert, qerr = q.GetCertificateByRecord(ctx, recordID)
		return qerr
	})
	if err != nil {
		return "", "", apperror.NewError("no certificate available for this record").AddError(err)
	}
	if cert.CertPath == "" || cert.KeyPath == "" {
		return "", "", apperror.NewError("certificate has not been issued yet")
	}

	certBytes, err := os.ReadFile(cert.CertPath)
	if err != nil {
		return "", "", apperror.NewError("failed to read certificate file").AddError(err)
	}
	keyBytes, err := os.ReadFile(cert.KeyPath)
	if err != nil {
		return "", "", apperror.NewError("failed to read private key file").AddError(err)
	}

	return string(certBytes), string(keyBytes), nil
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
