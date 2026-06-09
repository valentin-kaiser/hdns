package api

import (
	"bytes"
	"context"

	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/security"
	"github.com/valentin-kaiser/hdns/pkg/config"
	"github.com/valentin-kaiser/hdns/pkg/database"
	"github.com/valentin-kaiser/hdns/pkg/database/schema"
	"github.com/valentin-kaiser/hdns/pkg/dns"
	"github.com/valentin-kaiser/hdns/pkg/proto/service"
)

func (s *Server) GetZones(ctx context.Context, in *service.Record) (*service.ZoneList, error) {

	if in.Id != 0 && in.Token == "" {
		err := database.HDNS().Query(func(q *schema.Queries) error {
			var r *schema.Record
			r, err := q.GetRecord(ctx, in.Id)
			if err != nil {
				return apperror.NewError("failed to fetch record from database").AddError(err)
			}

			keyBytes := config.EncryptionKey()
			if len(keyBytes) != 32 {
				return apperror.NewError("invalid token encryption key")
			}

			var plainBuf bytes.Buffer
			cipher := security.NewAesCipher().WithPassphrase(keyBytes).Decrypt(r.Token, &plainBuf)
			if cipher.Error != nil {
				return apperror.NewError("failed to decrypt record token").AddError(cipher.Error)
			}

			in.Token = plainBuf.String()
			return nil
		})
		if err != nil {
			return nil, apperror.Wrap(err)
		}
	}

	zones, err := dns.FetchZones(ctx, in.Token)
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	list := &service.ZoneList{Zones: make([]*service.Zone, 0, len(zones))}
	for _, z := range zones {
		list.Zones = append(list.Zones, &service.Zone{
			Id:          z.ID,
			Name:        z.Name,
			RecordCount: int64(z.RecordCount),
		})
	}

	return list, nil
}
