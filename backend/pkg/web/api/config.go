package api

import (
	"context"

	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/hdns/pkg/config"
	"github.com/valentin-kaiser/hdns/pkg/proto/service"
)

func (s *Server) GetConfig(ctx context.Context, in *service.Empty) (*service.Configuration, error) {
	c := config.Get()
	return c.ToProto(), nil
}

func (s *Server) UpdateConfig(ctx context.Context, in *service.Configuration) (*service.Configuration, error) {
	current := config.Get()
	c := &config.App{
		LogLevel:        current.LogLevel,
		WebPort:         current.WebPort,
		CertificatePath: current.CertificatePath,
		KeyPath:         current.KeyPath,
		RefreshCron:     current.RefreshCron,
		DNSServers:      current.DNSServers,
		IPv4Resolvers:   current.IPv4Resolvers,
		IPv6Resolvers:   current.IPv6Resolvers,
		Database:        current.Database,
		// SMTP transport fields are not part of the proto and remain
		// unchanged on UpdateConfig; FromProto only overwrites the
		// Notifications subsection.
		Mail:          current.Mail,
		Notifications: current.Notifications,
	}

	err := c.FromProto(in).Validate()
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	err = config.Write(c)
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	return c.ToProto(), nil
}
