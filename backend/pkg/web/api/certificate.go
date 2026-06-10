package api

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/hdns/pkg/database"
	"github.com/valentin-kaiser/hdns/pkg/database/schema"
	"github.com/valentin-kaiser/hdns/pkg/dns"
	"github.com/valentin-kaiser/hdns/pkg/proto/service"
	"github.com/valentin-kaiser/hdns/pkg/tasks"
)

// certificateToProto converts a stored certificate into its proto
// representation.
func certificateToProto(c *schema.Certificate) *service.Certificate {
	if c == nil {
		return nil
	}
	proto := &service.Certificate{
		Id:        c.ID,
		CreatedAt: c.CreatedAt.Time.UnixMilli(),
		UpdatedAt: c.UpdatedAt.Time.UnixMilli(),
		RecordId:  c.RecordID,
		Domains:   c.Domains,
		Status:    c.Status,
		Serial:    c.Serial.String,
		LastError: c.LastError.String,
	}
	if c.NotBefore.Valid {
		proto.NotBefore = c.NotBefore.Time.UnixMilli()
	}
	if c.NotAfter.Valid {
		proto.NotAfter = c.NotAfter.Time.UnixMilli()
	}
	return proto
}

func (s *Server) GetCertificates(ctx context.Context, _ *service.Empty) (*service.CertificateList, error) {
	var certificates []*schema.Certificate
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var qerr error
		certificates, qerr = q.ListCertificates(ctx)
		return qerr
	})
	if err != nil {
		return nil, apperror.NewError("failed to fetch certificates from database").AddError(err)
	}

	list := &service.CertificateList{}
	for _, c := range certificates {
		list.Certificates = append(list.Certificates, certificateToProto(c))
	}
	return list, nil
}

func (s *Server) IssueCertificate(ctx context.Context, in *service.Record) (*service.Certificate, error) {
	if in == nil || in.Id == 0 {
		return nil, apperror.NewError("record id is required")
	}

	if err := dns.StartIssuance(in.Id); err != nil {
		return nil, apperror.NewError("failed to issue certificate").AddError(err)
	}

	var cert *schema.Certificate
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var qerr error
		cert, qerr = q.GetCertificateByRecord(ctx, in.Id)
		return qerr
	})
	if err != nil {
		return nil, apperror.NewError("failed to fetch certificate from database").AddError(err)
	}

	return certificateToProto(cert), nil
}

func (s *Server) GetCertificateDetails(ctx context.Context, in *service.Record) (*service.CertificateDetails, error) {
	if in == nil || in.Id == 0 {
		return nil, apperror.NewError("record id is required")
	}

	var record *schema.Record
	var cert *schema.Certificate
	var jobs []*schema.CertificateJob
	var runs []*schema.TaskRun
	var recordTasks []*schema.Task
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var qerr error
		record, qerr = q.GetRecord(ctx, in.Id)
		if qerr != nil {
			return qerr
		}
		cert, qerr = q.GetCertificateByRecord(ctx, in.Id)
		if qerr != nil && !errors.Is(qerr, sql.ErrNoRows) {
			return qerr
		}
		if errors.Is(qerr, sql.ErrNoRows) {
			cert = nil
		}
		jobs, qerr = q.ListCertificateJobsByRecord(ctx, in.Id)
		if qerr != nil {
			return qerr
		}
		runs, qerr = q.ListTaskRunsByRecord(ctx, in.Id)
		if qerr != nil {
			return qerr
		}
		recordTasks, qerr = q.ListTasksByRecord(ctx, in.Id)
		return qerr
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.NewError("record not found")
		}
		return nil, apperror.NewError("failed to fetch certificate details").AddError(err)
	}

	taskNameByID := make(map[int64]string, len(recordTasks))
	for _, t := range recordTasks {
		taskNameByID[t.ID] = t.Name
	}

	details := &service.CertificateDetails{
		RecordId:     record.ID,
		RecordName:   record.Name,
		RecordDomain: record.Domain,
		Certificate:  certificateToProto(cert),
		Artifacts:    buildProtoArtifacts(record.ID, cert),
		IssuanceJobs: mapProtoIssuanceJobs(jobs),
		TaskRuns:     mapProtoTaskRuns(runs, taskNameByID),
		FetchedAt:    time.Now().UnixMilli(),
	}
	return details, nil
}

func buildProtoArtifacts(recordID int64, cert *schema.Certificate) []*service.CertificateArtifact {
	keys := []struct{ key, label string }{
		{"cert", "cert.pem"},
		{"chain", "chain.pem"},
		{"fullchain", "fullchain.pem"},
		{"privkey", "privkey.pem"},
		{"pkcs12", "certificate.p12"},
	}
	artifacts := make([]*service.CertificateArtifact, 0, len(keys))
	var available map[string]bool
	if cert != nil && cert.CertPath != "" && cert.KeyPath != "" {
		certDir := filepath.Dir(cert.CertPath)
		available = map[string]bool{
			"cert":      fileExists(filepath.Join(certDir, "cert.pem")),
			"chain":     fileExists(filepath.Join(certDir, "chain.pem")),
			"fullchain": fileExists(cert.CertPath),
			"privkey":   fileExists(cert.KeyPath),
		}
		available["pkcs12"] = available["fullchain"] && available["privkey"]
	}
	for _, k := range keys {
		a := &service.CertificateArtifact{
			Key:         k.key,
			Label:       k.label,
			DownloadUrl: "/api/certificates/download?record_id=" + strconv.FormatInt(recordID, 10) + "&artifact=" + k.key,
		}
		if available != nil {
			a.Available = available[k.key]
		}
		artifacts = append(artifacts, a)
	}
	return artifacts
}

func mapProtoIssuanceJobs(jobs []*schema.CertificateJob) []*service.CertificateIssuanceJob {
	out := make([]*service.CertificateIssuanceJob, 0, len(jobs))
	for _, job := range jobs {
		entry := &service.CertificateIssuanceJob{
			Id:        job.ID,
			Source:    job.Source,
			Status:    job.Status,
			StartedAt: job.StartedAt.UnixMilli(),
			Error:     job.Error.String,
		}
		if job.FinishedAt.Valid {
			entry.FinishedAt = job.FinishedAt.Time.UnixMilli()
			entry.DurationMs = job.FinishedAt.Time.Sub(job.StartedAt).Milliseconds()
		}
		if job.CertificateID.Valid {
			entry.CertificateId = job.CertificateID.Int64
		}
		out = append(out, entry)
	}
	return out
}

func mapProtoTaskRuns(runs []*schema.TaskRun, taskNameByID map[int64]string) []*service.CertificateTaskRun {
	out := make([]*service.CertificateTaskRun, 0, len(runs))
	for _, run := range runs {
		if run.TriggerOn != int8(tasks.TriggerCert) {
			continue
		}
		entry := &service.CertificateTaskRun{
			Id:             run.ID,
			TaskId:         run.TaskID,
			TaskName:       taskNameByID[run.TaskID],
			Status:         run.Status,
			ResponseStatus: run.ResponseStatus.String,
			Error:          run.Error.String,
			StartedAt:      run.StartedAt.UnixMilli(),
		}
		if run.FinishedAt.Valid {
			entry.FinishedAt = run.FinishedAt.Time.UnixMilli()
			entry.DurationMs = run.FinishedAt.Time.Sub(run.StartedAt).Milliseconds()
		}
		if run.CertificateJobID.Valid {
			entry.CertificateJobId = run.CertificateJobID.Int64
		}
		out = append(out, entry)
	}
	return out
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func certificateFileBase(record *schema.Record) string {
	domain := strings.TrimSuffix(record.Domain, ".")
	switch record.Name {
	case "", "@":
		return strings.ReplaceAll(domain, "*", "wildcard")
	default:
		name := strings.ReplaceAll(record.Name, "*", "wildcard")
		return strings.ReplaceAll(name+"."+domain, "*", "wildcard")
	}
}
