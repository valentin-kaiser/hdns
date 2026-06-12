package api

import (
	"context"
	"strings"

	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/logging"
	"github.com/valentin-kaiser/hdns/pkg/proto/service"
)

func (s *Server) StreamLog(ctx context.Context, in *service.Empty, out chan<- *service.Line) error {
	adapter, ok := logging.GetGlobalAdapter[*logging.ZerologAdapter]()
	if !ok {
		return apperror.NewError("log streaming not supported")
	}

	writer := adapter.Stream()
	listener := make(chan string, 200)
	writer.AddListener(listener)
	defer writer.RemoveListener(listener)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case line := <-listener:
			line = strings.TrimRight(line, "\r\n")
			out <- &service.Line{Line: line}
		}
	}
}
