package dns

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/queue"
	"github.com/valentin-kaiser/hdns/pkg/config"
	"github.com/valentin-kaiser/hdns/pkg/database"
	"github.com/valentin-kaiser/hdns/pkg/database/schema"
)

var scheduler = queue.NewTaskScheduler()

func Start(ctx context.Context) error {
	err := scheduler.RegisterCronTask("ddns-refresh", config.Get().RefreshCron, Refresh)
	if err != nil {
		return apperror.NewError("failed to add cron job for DNS refresh").AddError(err)
	}

	err = scheduler.Start(ctx)
	if err != nil {
		return apperror.NewError("failed to start DNS refresh cron job").AddError(err)
	}

	return nil
}

func Stop() {
	scheduler.Stop()
}

func Restart(ctx context.Context) error {
	Stop()
	err := Start(ctx)
	if err != nil {
		return apperror.Wrap(err)
	}
	log.Info().Msg("refresh cron job restarted")
	return nil
}

func Refresh(ctx context.Context) error {
	_, err := UpdateAddress(ctx)
	if err != nil {
		return apperror.NewError("failed to update public IP address").AddError(err)
	}
	var records []*schema.Record
	err = database.HDNS().Query(func(q *schema.Queries) error {
		records, err = q.ListRecords(context.Background())
		if err != nil {
			return apperror.Wrap(err)
		}

		return nil
	})
	if err != nil {
		return apperror.NewError("failed to fetch DNS records").AddError(err)
	}
	for _, record := range records {
		err := RefreshRecord(ctx, record)
		if err != nil {
			log.Error().Err(err).Msgf("failed to refresh DNS record %s.%s", record.Name, record.Domain)
		}
	}

	return nil
}

func RefreshRecord(ctx context.Context, record *schema.Record) error {
	var current *schema.Address
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var err error
		current, err = q.GetCurrentAddress(context.Background())
		if err != nil {
			return apperror.Wrap(err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	rec, found, err := FetchRecord(record)
	if err != nil {
		return err
	}

	if found && rec.Value == current.Ipv4.String {
		log.Info().Msgf("record %s.%s is already up-to-date with address %s", record.Name, record.Domain, current.Ipv4.String)
		return nil
	}

	err = UpdateRecord(ctx, record, current)
	if err != nil {
		return err
	}
	return nil
}
