package api

import (
	"context"
	"strings"
	"time"

	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/hdns/pkg/database"
	"github.com/valentin-kaiser/hdns/pkg/database/schema"
	"github.com/valentin-kaiser/hdns/pkg/dns"
	"github.com/valentin-kaiser/hdns/pkg/proto/service"
)

func (s *Server) GetRecords(ctx context.Context, _ *service.Empty) (*service.RecordList, error) {
	var records []*schema.Record
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var err error
		records, err = q.ListRecords(ctx)
		if err != nil {
			return apperror.NewError("failed to fetch records from database").AddError(err)
		}
		return nil
	})
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	list := &service.RecordList{}
	for _, record := range records {
		proto := &service.Record{
			Id:          record.ID,
			CreatedAt:   record.CreatedAt.Time.UnixMilli(),
			UpdatedAt:   record.UpdatedAt.Time.UnixMilli(),
			Token:       record.Token,
			ZoneId:      record.ZoneID,
			Domain:      record.Domain,
			Name:        record.Name,
			Ttl:         uint32(record.Ttl),
			AddressId:   record.AddressID.Int64,
			LastRefresh: record.LastRefresh.Time.UnixMilli(),
		}

		if record.AddressID.Valid {
			var address *schema.Address
			err := database.HDNS().Query(func(q *schema.Queries) error {
				var err error
				address, err = q.GetAddressByID(ctx, record.AddressID.Int64)
				if err != nil {
					return apperror.NewError("failed to fetch address from database").AddError(err)
				}
				return nil
			})
			if err != nil {
				return nil, apperror.Wrap(err)
			}

			proto.Address = &service.Address{
				Id:        address.ID,
				CreatedAt: address.CreatedAt.Time.UnixMilli(),
				UpdatedAt: address.UpdatedAt.Time.UnixMilli(),
				Ipv4:      address.Ipv4.String,
				Ipv6:      address.Ipv6.String,
				Current:   address.Current,
			}
		}

		list.Records = append(list.Records, proto)
	}
	return list, nil
}

func (s *Server) UpsertRecord(ctx context.Context, in *service.Record) (*service.Record, error) {
	if in == nil {
		return nil, apperror.NewError("record is required")
	}

	if strings.TrimSpace(in.Token) == "" {
		return nil, apperror.NewError("record token is required")
	}

	if strings.TrimSpace(in.ZoneId) == "" {
		return nil, apperror.NewError("zone ID is required")
	}

	if strings.TrimSpace(in.Domain) == "" {
		return nil, apperror.NewError("record domain is required")
	}

	if strings.TrimSpace(in.Name) == "" {
		return nil, apperror.NewError("record name is required")
	}

	var record *schema.Record
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var err error
		switch in.Id {
		case 0:
			in.Id, err = q.CreateRecord(ctx, schema.CreateRecordParams{
				Token:  in.Token,
				ZoneID: in.ZoneId,
				Domain: in.Domain,
				Name:   in.Name,
				Ttl:    int32(in.Ttl),
			})
			if err != nil {
				return apperror.NewError("failed to create record in database").AddError(err)
			}
		default:
			_, err := q.UpdateRecord(ctx, schema.UpdateRecordParams{
				ID:     in.Id,
				Token:  in.Token,
				ZoneID: in.ZoneId,
				Domain: in.Domain,
				Name:   in.Name,
				Ttl:    int32(in.Ttl),
			})
			if err != nil {
				return apperror.NewError("failed to update record in database").AddError(err)
			}
		}

		record, err = q.GetRecord(ctx, in.Id)
		if err != nil {
			return apperror.NewError("failed to fetch record from database").AddError(err)
		}

		return nil
	})
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	err = dns.RefreshRecord(ctx, record)
	if err != nil {
		return nil, apperror.NewError("failed to refresh record").AddError(err)
	}

	proto := &service.Record{
		Id:          record.ID,
		CreatedAt:   record.CreatedAt.Time.UnixMilli(),
		UpdatedAt:   record.UpdatedAt.Time.UnixMilli(),
		Token:       record.Token,
		ZoneId:      record.ZoneID,
		Domain:      record.Domain,
		Name:        record.Name,
		Ttl:         uint32(record.Ttl),
		AddressId:   record.AddressID.Int64,
		LastRefresh: record.LastRefresh.Time.UnixMilli(),
	}

	if record.AddressID.Valid {
		var address *schema.Address
		err := database.HDNS().Query(func(q *schema.Queries) error {
			var err error
			address, err = q.GetAddressByID(ctx, record.AddressID.Int64)
			if err != nil {
				return apperror.NewError("failed to fetch address from database").AddError(err)
			}
			return nil
		})
		if err != nil {
			return nil, apperror.Wrap(err)
		}

		proto.Address = &service.Address{
			Id:        address.ID,
			CreatedAt: address.CreatedAt.Time.UnixMilli(),
			UpdatedAt: address.UpdatedAt.Time.UnixMilli(),
			Ipv4:      address.Ipv4.String,
			Ipv6:      address.Ipv6.String,
			Current:   address.Current,
		}
	}

	return proto, nil
}

func (s *Server) DeleteRecord(ctx context.Context, in *service.RecordDelete) (*service.Empty, error) {
	if in == nil || in.Record == nil {
		return nil, apperror.NewError("record is required")
	}

	err := database.HDNS().Query(func(q *schema.Queries) error {
		record, err := q.GetRecord(ctx, in.Record.Id)
		if err != nil {
			return apperror.NewError("failed to fetch record from database").AddError(err)
		}

		if in.DeleteFromHetzner {
			err = dns.DeleteRecord(ctx, record)
			if err != nil {
				return apperror.NewError("failed to delete record from Hetzner DNS").AddError(err)
			}
		}

		err = q.DeleteRecord(ctx, in.Record.Id)
		if err != nil {
			return apperror.NewError("failed to delete record from database").AddError(err)
		}
		return nil
	})
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	return &service.Empty{}, nil
}

func (s *Server) RefreshRecord(ctx context.Context, in *service.Record) (*service.Record, error) {
	var address *schema.Address
	var record *schema.Record
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var err error
		record, err = q.GetRecord(ctx, in.Id)
		if err != nil {
			return apperror.NewError("failed to fetch record from database").AddError(err)
		}

		address, err = dns.UpdateAddress(ctx)
		if err != nil {
			return apperror.NewError("failed to update public IP address").AddError(err)
		}

		err = dns.UpdateRecord(ctx, record, address)
		if err != nil {
			return apperror.NewError("failed to update DNS record").AddError(err)
		}

		return nil
	})
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	proto := &service.Record{
		Id:          record.ID,
		CreatedAt:   record.CreatedAt.Time.UnixMilli(),
		UpdatedAt:   record.UpdatedAt.Time.UnixMilli(),
		Token:       record.Token,
		ZoneId:      record.ZoneID,
		Domain:      record.Domain,
		Name:        record.Name,
		Ttl:         uint32(record.Ttl),
		AddressId:   record.AddressID.Int64,
		LastRefresh: record.LastRefresh.Time.UnixMilli(),
	}

	if record.AddressID.Valid {
		proto.Address = &service.Address{
			Id:        address.ID,
			CreatedAt: address.CreatedAt.Time.UnixMilli(),
			UpdatedAt: address.UpdatedAt.Time.UnixMilli(),
			Ipv4:      address.Ipv4.String,
			Ipv6:      address.Ipv6.String,
			Current:   address.Current,
		}
	}

	return proto, nil
}

func (s *Server) ResolveRecord(ctx context.Context, in *service.Record) (*service.ResolutionResult, error) {
	if in == nil {
		return nil, apperror.NewError("record is required")
	}

	var record *schema.Record
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var err error
		record, err = q.GetRecord(ctx, in.Id)
		if err != nil {
			return apperror.NewError("failed to fetch record from database").AddError(err)
		}
		return nil
	})
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	resolver := dns.NewDNSResolver()
	domain := resolver.BuildDomain(record)
	result, err := resolver.Resolve(domain)
	if err != nil {
		return nil, apperror.NewError("failed to resolve record").AddError(err)
	}

	list := &service.ResolutionResult{Resolutions: make([]*service.Resolution, 0)}
	for _, res := range result {
		list.Resolutions = append(list.Resolutions, &service.Resolution{
			Server:       res.Server,
			Addresses:    res.Addresses,
			ResponseTime: res.ResponseTime,
			Error:        res.Error,
		})
	}

	return list, nil
}

func (s *Server) StreamResolveRecord(ctx context.Context, in *service.Record, out chan<- *service.Resolution) error {
	if in == nil {
		return apperror.NewError("record is required")
	}

	var record *schema.Record
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var err error
		record, err = q.GetRecord(ctx, in.Id)
		if err != nil {
			return apperror.NewError("failed to fetch record from database").AddError(err)
		}
		return nil
	})
	if err != nil {
		return apperror.Wrap(err)
	}

	resolver := dns.NewDNSResolver()
	domain := resolver.BuildDomain(record)
	for {
		result, err := resolver.Resolve(domain)
		if err != nil {
			return apperror.NewError("failed to resolve record").AddError(err)
		}

		for _, res := range result {
			out <- &service.Resolution{
				Server:       res.Server,
				Addresses:    res.Addresses,
				ResponseTime: res.ResponseTime,
				Error:        res.Error,
			}
		}
		time.Sleep(5 * time.Second)
	}
}
