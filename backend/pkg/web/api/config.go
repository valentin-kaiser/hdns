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
	c := &config.App{
		LogLevel:        config.Get().LogLevel,
		WebPort:         config.Get().WebPort,
		CertificatePath: config.Get().CertificatePath,
		KeyPath:         config.Get().KeyPath,
		RefreshCron:     config.Get().RefreshCron,
		DNSServers:      config.Get().DNSServers,
		IPv4Resolvers:   config.Get().IPv4Resolvers,
		IPv6Resolvers:   config.Get().IPv6Resolvers,
		Database:        config.Get().Database,
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
