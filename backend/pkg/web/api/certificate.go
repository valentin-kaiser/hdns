package api

import (
	"context"

	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/hdns/pkg/database"
	"github.com/valentin-kaiser/hdns/pkg/database/schema"
	"github.com/valentin-kaiser/hdns/pkg/dns"
	"github.com/valentin-kaiser/hdns/pkg/proto/service"
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
