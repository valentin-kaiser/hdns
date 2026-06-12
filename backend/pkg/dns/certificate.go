package dns

import (
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-acme/lego/v5/certificate"
	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/flag"
	"github.com/valentin-kaiser/go-core/logging/log"
	"github.com/valentin-kaiser/hdns/pkg/config"
	"github.com/valentin-kaiser/hdns/pkg/database"
	"github.com/valentin-kaiser/hdns/pkg/database/schema"
	"github.com/valentin-kaiser/hdns/pkg/tasks"
)

// pending tracks records whose certificate is currently being issued
// so concurrent requests cannot trigger overlapping ACME orders for the same
// record, which the ACME server rejects with "orderNotReady".
var pending sync.Map

// Record purpose values, mirroring the records.purpose column.
const (
	PurposeDDNS int8 = 1
	PurposeCert int8 = 2
	PurposeBoth int8 = 3
)

// IssuanceSource identifies where a certificate issuance was triggered.
type IssuanceSource string

const (
	IssuanceSourceManual    IssuanceSource = "manual"
	IssuanceSourceScheduled IssuanceSource = "scheduled"
)

// recordDoesDDNS reports whether a record should have its A record kept in
// sync with the public IP address.
func recordDoesDDNS(purpose int8) bool {
	return purpose == PurposeDDNS || purpose == PurposeBoth
}

// recordDoesCert reports whether a record should have a certificate issued.
func recordDoesCert(purpose int8) bool {
	return purpose == PurposeCert || purpose == PurposeBoth
}

// IssueCertificate issues (or re-issues) a certificate for the record with the
// given ID.
func IssueCertificate(ctx context.Context, recordID int64) error {
	var record *schema.Record
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var qerr error
		record, qerr = q.GetRecord(ctx, recordID)
		return qerr
	})
	if err != nil {
		return apperror.NewError("failed to load record").AddError(err)
	}
	return issueForRecord(ctx, record, IssuanceSourceManual)
}

// StartIssuance validates the record, ensures a certificate row exists so the
// UI immediately reflects the pending state, and runs the (long-running) ACME
// flow in a background goroutine. It returns as soon as issuance has been
// scheduled so the HTTP request does not block for the duration of the ACME
// challenge, which avoids client timeouts and duplicate requests.
func StartIssuance(recordID int64) error {
	var record *schema.Record
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var qerr error
		record, qerr = q.GetRecord(context.Background(), recordID)
		return qerr
	})
	if err != nil {
		return apperror.NewError("failed to load record").AddError(err)
	}

	if !recordDoesCert(record.Purpose) {
		return apperror.NewError("record is not configured for certificate issuance")
	}
	if !config.Get().ACME.Enabled {
		return apperror.NewError("ACME is not enabled")
	}

	// Reserve the per-record slot synchronously so a duplicate request is
	// rejected immediately instead of after the ACME flow has started.
	if _, loaded := pending.LoadOrStore(record.ID, struct{}{}); loaded {
		return apperror.NewError("certificate issuance is already in progress for this record")
	}

	// Make the pending certificate row visible to the UI before returning.
	if _, err := ensureCertificateRow(context.Background(), record, buildDomains(record)); err != nil {
		pending.Delete(record.ID)
		return apperror.Wrap(err)
	}

	go func() {
		defer pending.Delete(record.ID)
		if err := runIssuance(context.Background(), record, IssuanceSourceManual); err != nil {
			log.Error().Err(err).Msgf("[ACME] background issuance failed for %s.%s", record.Name, record.Domain)
		}
	}()

	return nil
}

// issueForRecord acquires the per-record in-flight lock and runs the ACME flow
// synchronously. It is used by callers that need to wait for completion, such
// as the renewal job.
func issueForRecord(ctx context.Context, record *schema.Record, source IssuanceSource) error {
	if !recordDoesCert(record.Purpose) {
		return apperror.NewError("record is not configured for certificate issuance")
	}
	if !config.Get().ACME.Enabled {
		return apperror.NewError("ACME is not enabled")
	}

	// Serialize issuance per record: overlapping ACME orders for the same
	// identifiers are rejected by the ACME server with "orderNotReady".
	if _, loaded := pending.LoadOrStore(record.ID, struct{}{}); loaded {
		return apperror.NewError("certificate issuance is already in progress for this record")
	}
	defer pending.Delete(record.ID)

	// Detach from the caller's context so a cancelled HTTP request cannot abort
	// a half-completed issuance (the ACME flow can take ~30s and the resulting
	// certificate must always be persisted once obtained).
	return runIssuance(context.WithoutCancel(ctx), record, source)
}

// runIssuance performs the actual ACME flow, persists the resulting certificate
// to disk and database, and fires certificate renewal tasks. Callers must hold
// the per-record in-flight lock and pass a context that is not tied to a
// cancellable request.
func runIssuance(ctx context.Context, record *schema.Record, source IssuanceSource) error {
	domains := buildDomains(record)

	certID, err := ensureCertificateRow(ctx, record, domains)
	if err != nil {
		return apperror.Wrap(err)
	}

	jobID, err := createCertificateJob(ctx, record.ID, certID, source)
	if err != nil {
		return apperror.Wrap(err)
	}

	fail := func(cause error) error {
		setCertificateFailed(ctx, certID, cause)
		finishCertificateJob(ctx, jobID, "failed", cause)
		return apperror.Wrap(cause)
	}

	res, err := obtainCertificate(record, domains, record.Domain)
	if err != nil {
		return fail(err)
	}

	certPath, keyPath, err := writeCertificateFiles(record, res)
	if err != nil {
		return fail(err)
	}

	leaf, err := parseLeafCertificate(res.Certificate)
	if err != nil {
		return fail(err)
	}

	err = database.HDNS().Query(func(q *schema.Queries) error {
		return q.UpdateCertificate(ctx, schema.UpdateCertificateParams{
			ID:        certID,
			Domains:   strings.Join(domains, ","),
			Status:    "valid",
			NotBefore: sql.NullTime{Time: leaf.NotBefore, Valid: true},
			NotAfter:  sql.NullTime{Time: leaf.NotAfter, Valid: true},
			Serial:    sql.NullString{String: leaf.SerialNumber.String(), Valid: true},
			LastError: sql.NullString{},
			CertPath:  certPath,
			KeyPath:   keyPath,
		})
	})
	if err != nil {
		return fail(apperror.NewError("failed to persist certificate").AddError(err))
	}

	finishCertificateJob(ctx, jobID, "success", nil)
	tasks.FireTasks(ctx, record.ID, tasks.TriggerCert, sql.NullInt64{Int64: jobID, Valid: true})
	return nil
}

// RenewCertificates re-issues all certificates that are due for renewal based
// on the configured renewal window.
func RenewCertificates(ctx context.Context) error {
	cfg := config.Get()
	if !cfg.ACME.Enabled {
		return nil
	}

	threshold := time.Now().Add(time.Duration(cfg.ACME.RenewBeforeDays) * 24 * time.Hour)

	var certs []*schema.Certificate
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var qerr error
		certs, qerr = q.ListCertificatesForRenewal(ctx, sql.NullTime{Time: threshold, Valid: true})
		return qerr
	})
	if err != nil {
		return apperror.NewError("failed to list certificates for renewal").AddError(err)
	}

	for _, cert := range certs {
		var record *schema.Record
		lerr := database.HDNS().Query(func(q *schema.Queries) error {
			var qerr error
			record, qerr = q.GetRecord(ctx, cert.RecordID)
			return qerr
		})
		if lerr != nil {
			log.Error().Err(lerr).Msgf("[ACME] failed to load record %d for renewal", cert.RecordID)
			continue
		}
		if !recordDoesCert(record.Purpose) {
			continue
		}
		if ierr := issueForRecord(ctx, record, IssuanceSourceScheduled); ierr != nil {
			log.Error().Err(ierr).Msgf("[ACME] failed to renew certificate for %s.%s", record.Name, record.Domain)
		}
	}
	return nil
}

func createCertificateJob(ctx context.Context, recordID, certID int64, source IssuanceSource) (int64, error) {
	var jobID int64
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var qerr error
		jobID, qerr = q.CreateCertificateJob(ctx, schema.CreateCertificateJobParams{
			RecordID:      recordID,
			CertificateID: sql.NullInt64{Int64: certID, Valid: certID != 0},
			Source:        string(source),
			Status:        "running",
			StartedAt:     time.Now(),
		})
		return qerr
	})
	if err != nil {
		return 0, apperror.NewError("failed to create certificate job history entry").AddError(err)
	}
	return jobID, nil
}

func finishCertificateJob(ctx context.Context, jobID int64, status string, runErr error) {
	params := schema.FinishCertificateJobParams{
		ID:         jobID,
		Status:     status,
		FinishedAt: sql.NullTime{Time: time.Now(), Valid: true},
	}
	if runErr != nil {
		params.Error = sql.NullString{String: runErr.Error(), Valid: true}
	}

	err := database.HDNS().Query(func(q *schema.Queries) error {
		return q.FinishCertificateJob(ctx, params)
	})
	if err != nil {
		log.Error().Err(err).Msgf("[ACME] failed to update certificate job history %d", jobID)
	}
}

// ensureCertificateRow returns the certificate row id for the record, creating
// a pending row if none exists.
func ensureCertificateRow(ctx context.Context, record *schema.Record, domains []string) (int64, error) {
	var certID int64
	err := database.HDNS().Query(func(q *schema.Queries) error {
		existing, gerr := q.GetCertificateByRecord(ctx, record.ID)
		if gerr == nil {
			certID = existing.ID
			return nil
		}
		if !errors.Is(gerr, sql.ErrNoRows) {
			return gerr
		}

		id, cerr := q.CreateCertificate(ctx, schema.CreateCertificateParams{
			RecordID: record.ID,
			Domains:  strings.Join(domains, ","),
			Status:   "pending",
		})
		if cerr != nil {
			return cerr
		}
		certID = id
		return nil
	})
	if err != nil {
		return 0, apperror.NewError("failed to prepare certificate record").AddError(err)
	}
	return certID, nil
}

// setCertificateFailed marks a certificate row as failed and records the error.
func setCertificateFailed(ctx context.Context, certID int64, cause error) {
	err := database.HDNS().Query(func(q *schema.Queries) error {
		return q.UpdateCertificateStatus(ctx, schema.UpdateCertificateStatusParams{
			ID:        certID,
			Status:    "failed",
			LastError: sql.NullString{String: cause.Error(), Valid: true},
		})
	})
	if err != nil {
		log.Error().Err(err).Msgf("[ACME] failed to mark certificate %d as failed", certID)
	}
}

// buildDomains derives the certificate SANs from the record, optionally adding
// a wildcard entry.
func buildDomains(record *schema.Record) []string {
	domain := strings.TrimSuffix(record.Domain, ".")

	var primary string
	switch record.Name {
	case "@", "":
		primary = domain
	case "*":
		primary = "*." + domain
	default:
		primary = record.Name + "." + domain
	}

	domains := []string{primary}
	if record.IncludeWildcard {
		wildcard := "*." + strings.TrimPrefix(primary, "*.")
		if wildcard != primary {
			domains = append(domains, wildcard)
		}
	}
	return domains
}

// certificateDir returns the on-disk directory used to store a record's
// certificate files.
func certificateDir(record *schema.Record) string {
	base := strings.TrimPrefix(buildDomains(record)[0], "*.")
	base = strings.ReplaceAll(base, "*", "wildcard")
	return filepath.Join(flag.Path, "certs", base)
}

// writeCertificateFiles persists the issued certificate material to disk,
// mirroring the layout produced by certbot:
//
//	cert.pem      – leaf/domain certificate only
//	chain.pem     – intermediate certificate(s) only
//	fullchain.pem – cert.pem + chain.pem (leaf followed by intermediates)
//	privkey.pem   – private key
//
// The function returns the paths to fullchain.pem and privkey.pem, which are
// stored in the database and used by the task webhook system.
func writeCertificateFiles(record *schema.Record, res *certificate.Resource) (string, string, error) {
	dir := certificateDir(record)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", apperror.NewError("failed to create certificate directory").AddError(err)
	}

	files := []struct {
		name    string
		content []byte
	}{
		{"cert.pem", res.Certificate},
		{"chain.pem", res.IssuerCertificate},
		{"fullchain.pem", append(res.Certificate, res.IssuerCertificate...)},
		{"privkey.pem", res.PrivateKey},
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f.name), f.content, 0o600); err != nil {
			return "", "", apperror.NewError("failed to write " + f.name).AddError(err)
		}
	}

	return filepath.Join(dir, "fullchain.pem"), filepath.Join(dir, "privkey.pem"), nil
}

// parseLeafCertificate extracts the leaf certificate from a PEM-encoded chain.
func parseLeafCertificate(pemData []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, apperror.NewError("failed to decode certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, apperror.NewError("failed to parse certificate").AddError(err)
	}
	return cert, nil
}
