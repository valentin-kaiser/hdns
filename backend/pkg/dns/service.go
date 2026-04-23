package dns

import (
	"context"

	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/logging/log"
	"github.com/valentin-kaiser/go-core/queue"
	"github.com/valentin-kaiser/hdns/pkg/config"
	"github.com/valentin-kaiser/hdns/pkg/database"
	"github.com/valentin-kaiser/hdns/pkg/database/schema"
	mailpkg "github.com/valentin-kaiser/hdns/pkg/mail"
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
	report := mailpkg.Report{}
	address, err := UpdateAddress(ctx)
	if err != nil {
		report.AddressError = err.Error()
		report.Severity = mailpkg.SeverityFailure
		mailpkg.SendReport(ctx, report)
		return apperror.NewError("failed to update public IP address").AddError(err)
	}
	if address != nil {
		if address.Ipv4.Valid {
			report.CurrentIPv4 = address.Ipv4.String
		}
		if address.Ipv6.Valid {
			report.CurrentIPv6 = address.Ipv6.String
		}
	}

	var records []*schema.Record
	err = database.HDNS().Query(func(q *schema.Queries) error {
		records, err = q.ListRecords(ctx, schema.ListRecordsParams{})
		if err != nil {
			return apperror.Wrap(err)
		}

		return nil
	})
	if err != nil {
		report.AddressError = err.Error()
		report.Severity = mailpkg.SeverityFailure
		mailpkg.SendReport(ctx, report)
		return apperror.NewError("failed to fetch DNS records").AddError(err)
	}
	for _, record := range records {
		status := refreshRecordStatus(ctx, record, address)
		switch status.Result {
		case "updated":
			report.UpdatedCount++
		case "failed":
			report.FailedCount++
			log.Error().Msgf("failed to refresh DNS record %s.%s: %s", record.Name, record.Domain, status.Error)
		default:
			report.UnchangedCount++
		}
		report.Records = append(report.Records, status)
	}

	switch {
	case report.FailedCount > 0 || report.AddressError != "":
		report.Severity = mailpkg.SeverityFailure
		mailpkg.SendReport(ctx, report)
	case report.UpdatedCount > 0:
		report.Severity = mailpkg.SeveritySuccess
		mailpkg.SendReport(ctx, report)
	}

	return nil
}

// refreshRecordStatus refreshes a single record and returns a structured
// status suitable for inclusion in a refresh report. It does not return an
// error separately: any error is captured in the returned RecordStatus.
func refreshRecordStatus(ctx context.Context, record *schema.Record, address *schema.Address) mailpkg.RecordStatus {
	status := mailpkg.RecordStatus{
		Domain: record.Domain,
		Name:   record.Name,
		Type:   "A",
	}

	if address == nil || !address.Ipv4.Valid {
		status.Result = "failed"
		status.Error = "current address has no valid IPv4; skipping A record update"
		return status
	}

	rrset, found, err := FetchRecord(ctx, record)
	if err != nil {
		status.Result = "failed"
		status.Error = err.Error()
		return status
	}
	if found && len(rrset.Records) > 0 {
		status.Value = rrset.Records[0].Value
		if rrset.Records[0].Value == address.Ipv4.String {
			status.Result = "unchanged"
			log.Info().Msgf("record %s.%s is already up-to-date with address %s", record.Name, record.Domain, address.Ipv4.String)
			return status
		}
	}

	if err := UpdateRecord(ctx, record, address); err != nil {
		status.Result = "failed"
		status.Error = err.Error()
		return status
	}
	status.Result = "updated"
	return status
}

func RefreshRecord(ctx context.Context, record *schema.Record) error {
	var current *schema.Address
	err := database.HDNS().Query(func(q *schema.Queries) error {
		var err error
		current, err = q.GetCurrentAddress(ctx)
		if err != nil {
			return apperror.Wrap(err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	if !current.Ipv4.Valid {
		return apperror.NewError("current address has no valid IPv4; skipping A record update")
	}

	rrset, found, err := FetchRecord(ctx, record)
	if err != nil {
		return err
	}

	if found && len(rrset.Records) > 0 && rrset.Records[0].Value == current.Ipv4.String {
		log.Info().Msgf("record %s.%s is already up-to-date with address %s", record.Name, record.Domain, current.Ipv4.String)
		return nil
	}

	err = UpdateRecord(ctx, record, current)
	if err != nil {
		return err
	}
	return nil
}
