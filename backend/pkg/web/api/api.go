package api

import (
	"github.com/valentin-kaiser/hdns/pkg/proto/service"
)

type Server struct {
	service.UnimplementedHDNSServer
}
