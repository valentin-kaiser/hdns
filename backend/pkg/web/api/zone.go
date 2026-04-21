package api

import (
	"context"
	"strconv"

	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/hdns/pkg/dns"
	"github.com/valentin-kaiser/hdns/pkg/proto/service"
)

func (s *Server) GetZones(ctx context.Context, in *service.Request) (*service.ZoneList, error) {
	zones, err := dns.FetchZones(ctx, in.Token)
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	list := &service.ZoneList{Zones: make([]*service.Zone, 0, len(zones))}
	for _, z := range zones {
		list.Zones = append(list.Zones, &service.Zone{
			Id:          strconv.FormatInt(z.ID, 10),
			Name:        z.Name,
			RecordCount: int64(z.RecordCount),
		})
	}

	return list, nil
}
