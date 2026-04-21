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
	c := (&config.App{}).FromProto(in)
	if c == nil {
		return nil, apperror.NewError("invalid configuration provided")
	}

	err := c.Validate()
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	err = config.Write(c)
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	return c.ToProto(), nil
}
