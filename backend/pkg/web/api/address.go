package api

import (
	"context"
	"time"

	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/hdns/pkg/database"
	"github.com/valentin-kaiser/hdns/pkg/database/schema"
	"github.com/valentin-kaiser/hdns/pkg/dns"
	"github.com/valentin-kaiser/hdns/pkg/proto/service"
)

func (s *Server) StreamAddress(ctx context.Context, in *service.Empty, out chan<- *service.Address) error {
	var current *schema.Address
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			var address *schema.Address
			err := database.HDNS().Query(func(q *schema.Queries) error {
				var err error
				address, err = q.GetCurrentAddress(ctx)
				if err != nil {
					return apperror.NewError("failed to fetch current address from database").AddError(err)
				}
				return nil
			})
			if err != nil {
				return apperror.Wrap(err)
			}

			if current != nil && current.ID == address.ID {
				time.Sleep(1 * time.Second)
				continue
			}
			current = address
			proto := &service.Address{
				Id:      address.ID,
				Current: address.Current,
			}

			if address.Ipv4.Valid {
				proto.Ipv4 = address.Ipv4.String
			}

			if address.Ipv6.Valid {
				proto.Ipv6 = address.Ipv6.String
			}

			if address.CreatedAt.Valid {
				proto.CreatedAt = address.CreatedAt.Time.UnixMilli()
			}

			if address.UpdatedAt.Valid {
				proto.UpdatedAt = address.UpdatedAt.Time.UnixMilli()
			}

			out <- proto
			time.Sleep(1 * time.Second)
		}
	}
}

func (s *Server) GetAddress(ctx context.Context, _ *service.Empty) (*service.Address, error) {
	var address *schema.Address
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var err error
		address, err = q.GetCurrentAddress(ctx)
		if err != nil {
			return apperror.NewError("failed to fetch current address from database").AddError(err)
		}
		return nil
	})
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	return &service.Address{
		Id:        address.ID,
		CreatedAt: address.CreatedAt.Time.UnixMilli(),
		UpdatedAt: address.UpdatedAt.Time.UnixMilli(),
		Ipv4:      address.Ipv4.String,
		Ipv6:      address.Ipv6.String,
		Current:   address.Current,
	}, nil
}

func (s *Server) GetAddressHistory(ctx context.Context, _ *service.Empty) (*service.AddressHistory, error) {
	var addresses []*schema.Address
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var err error
		addresses, err = q.ListAddresses(ctx)
		if err != nil {
			return apperror.NewError("failed to fetch address history from database").AddError(err)
		}
		return nil
	})
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	history := &service.AddressHistory{Addresses: make([]*service.Address, 0, len(addresses))}
	for _, addr := range addresses {
		history.Addresses = append(history.Addresses, &service.Address{
			Id:        addr.ID,
			CreatedAt: addr.CreatedAt.Time.UnixMilli(),
			UpdatedAt: addr.UpdatedAt.Time.UnixMilli(),
			Ipv4:      addr.Ipv4.String,
			Ipv6:      addr.Ipv6.String,
			Current:   addr.Current,
		})
	}

	return history, nil
}

func (s *Server) RefreshAddress(ctx context.Context, _ *service.Empty) (*service.Address, error) {
	address, err := dns.UpdateAddress(ctx)
	if err != nil {
		return nil, apperror.NewError("failed to refresh address").AddError(err)
	}

	proto := &service.Address{
		Id:      address.ID,
		Current: address.Current,
	}

	if address.Ipv4.Valid {
		proto.Ipv4 = address.Ipv4.String
	}

	if address.Ipv6.Valid {
		proto.Ipv6 = address.Ipv6.String
	}

	if address.CreatedAt.Valid {
		proto.CreatedAt = address.CreatedAt.Time.UnixMilli()
	}

	if address.UpdatedAt.Valid {
		proto.UpdatedAt = address.UpdatedAt.Time.UnixMilli()
	}

	return proto, nil
}
