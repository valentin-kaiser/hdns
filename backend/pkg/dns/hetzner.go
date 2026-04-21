package dns

import (
	"context"
	"database/sql"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/rs/zerolog/log"
	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/version"
	"github.com/valentin-kaiser/hdns/pkg/database"
	"github.com/valentin-kaiser/hdns/pkg/database/schema"
)

func newClient(token string) *hcloud.Client {
	return hcloud.NewClient(
		hcloud.WithToken(token),
		hcloud.WithApplication("hdns", version.GitTag),
	)
}

// FetchZones returns all DNS zones accessible with the given token.
func FetchZones(ctx context.Context, token string) ([]*hcloud.Zone, error) {
	zones, err := newClient(token).Zone.All(ctx)
	if err != nil {
		return nil, apperror.NewError("failed to fetch zones").AddError(err)
	}
	return zones, nil
}

// FetchRecord looks up the A RRSet for the given record in Hetzner.
func FetchRecord(ctx context.Context, r *schema.Record) (*hcloud.ZoneRRSet, bool, error) {
	c := newClient(r.Token)
	return findResourceRecordSet(ctx, c, r)
}

// UpdateRecord creates or overwrites the A RRSet for r with addr.Ipv4,
// then saves address_id + last_refresh to the DB.
func UpdateRecord(ctx context.Context, r *schema.Record, addr *schema.Address) error {
	c := newClient(r.Token)
	err := upsertResourceRecordSet(ctx, c, r, addr.Ipv4.String)
	if err != nil {
		return apperror.Wrap(err)
	}
	log.Info().Msgf("[DNS] record %s.%s updated successfully", r.Name, r.Domain)

	r.AddressID = sql.NullInt64{Int64: addr.ID, Valid: true}
	r.LastRefresh = sql.NullTime{Time: time.Now(), Valid: true}
	err = database.HDNS().Query(func(q *schema.Queries) error {
		err := q.UpdateRecordAddress(ctx, schema.UpdateRecordAddressParams{
			AddressID:   r.AddressID,
			LastRefresh: r.LastRefresh,
			ID:          r.ID,
		})
		if err != nil {
			return apperror.Wrap(err)
		}
		return nil
	})
	if err != nil {
		return apperror.NewErrorf("failed to update DNS record %s.%s in database", r.Name, r.Domain).AddError(err)
	}
	return nil
}

// DeleteRecord removes the A RRSet for r from Hetzner.
func DeleteRecord(ctx context.Context, r *schema.Record) error {
	c := newClient(r.Token)
	rrset, found, err := findResourceRecordSet(ctx, c, r)
	if err != nil {
		return apperror.Wrap(err)
	}
	if !found {
		return apperror.NewError("record not found")
	}
	_, _, err = c.Zone.DeleteRRSet(ctx, rrset)
	if err != nil {
		return apperror.NewError("failed to delete RRSet").AddError(err)
	}
	return nil
}

// findResourceRecordSet looks up the A RRSet for the given record in Hetzner and returns it.
func findResourceRecordSet(ctx context.Context, c *hcloud.Client, r *schema.Record) (*hcloud.ZoneRRSet, bool, error) {
	rrset, _, err := c.Zone.GetRRSetByNameAndType(ctx, &hcloud.Zone{ID: r.ZoneID}, r.Name, hcloud.ZoneRRSetTypeA)
	if err != nil {
		return nil, false, apperror.NewError("failed to fetch RRSet").AddError(err)
	}
	if rrset == nil {
		return nil, false, nil
	}
	return rrset, true, nil
}

// upsertResourceRecordSet creates or overwrites the A RRSet for r with the given IP address.
func upsertResourceRecordSet(ctx context.Context, c *hcloud.Client, r *schema.Record, ip string) error {
	rrset, found, err := findResourceRecordSet(ctx, c, r)
	if err != nil {
		return apperror.Wrap(err)
	}
	ttl := int(r.Ttl)
	records := []hcloud.ZoneRRSetRecord{{Value: ip}}

	if !found {
		_, _, err = c.Zone.CreateRRSet(ctx, &hcloud.Zone{ID: r.ZoneID}, hcloud.ZoneRRSetCreateOpts{
			Name:    r.Name,
			Type:    hcloud.ZoneRRSetTypeA,
			TTL:     &ttl,
			Records: records,
		})
		if err != nil {
			return apperror.NewError("failed to create RRSet").AddError(err)
		}
		return nil
	}

	_, _, err = c.Zone.SetRRSetRecords(ctx, rrset, hcloud.ZoneRRSetSetRecordsOpts{
		Records: records,
	})
	if err != nil {
		return apperror.NewError("failed to set RRSet records").AddError(err)
	}
	return nil
}
