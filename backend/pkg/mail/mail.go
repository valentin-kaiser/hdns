// Package mail wraps github.com/valentin-kaiser/go-core/mail to deliver DNS
// refresh reports via SMTP.
//
// SMTP transport settings come from the YAML configuration (config.Mail) and
// are NOT exposed through the web UI. Only the user-facing behavior toggles
// live in config.Notifications and can be edited from the frontend.
package mail

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"strings"
	"sync"
	"time"

	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/logging/log"
	"github.com/valentin-kaiser/go-core/mail"
	"github.com/valentin-kaiser/hdns/pkg/config"
)

//go:embed templates/*.html
var templatesFS embed.FS

const reportTemplateName = "templates/report.html"

var templateFuncs = template.FuncMap{
	"emptyDash":   emptyDash,
	"statusColor": statusColor,
}

// Severity classifies a refresh report for cooldown and logging purposes.
type Severity string

const (
	// SeverityFailure is reported when at least one record failed to update
	// or the public address lookup failed.
	SeverityFailure Severity = "failure"
	// SeveritySuccess is reported when at least one record was updated
	// without failures.
	SeveritySuccess Severity = "success"
)

// RecordStatus describes the outcome of a single record in a refresh run.
type RecordStatus struct {
	Domain string
	Name   string
	Type   string
	Result string // "updated" | "unchanged" | "failed"
	Value  string
	Error  string
}

// Report captures a full refresh run suitable for rendering as an email body.
type Report struct {
	Severity       Severity
	CurrentIPv4    string
	CurrentIPv6    string
	AddressError   string
	Records        []RecordStatus
	UpdatedCount   int
	FailedCount    int
	UnchangedCount int
}

var (
	mu         sync.Mutex
	manager    *mail.Manager
	lastSentAt = map[Severity]time.Time{}
)

// Start initialises the underlying mail manager if both the client and the
// notification feature are enabled in the current configuration.
//
// Start is a no-op (and returns nil) when notifications are disabled or the
// mail client is disabled; this is the expected "not configured" state and
// must not prevent the rest of hdns from starting.
func Start(_ context.Context) error {
	mu.Lock()
	defer mu.Unlock()

	if manager != nil {
		return nil
	}

	if !config.Get().Notifications.Enabled || !config.Get().Mail.Enabled {
		log.Debug().Msg("mail: notifications disabled, skipping mail manager start")
		return nil
	}

	m := mail.NewManager(&mail.Config{
		Client: config.Get().Mail,
		Templates: mail.TemplateConfig{
			Enabled:          true,
			DefaultTemplate:  reportTemplateName,
			FileSystem:       templatesFS,
			WithDefaultFuncs: true,
			GlobalFuncs:      templateFuncs,
		},
	}, nil)
	if err := m.Start(); err != nil {
		return apperror.NewError("failed to start mail manager").AddError(err)
	}

	manager = m
	log.Info().Msg("mail: manager started")
	return nil
}

// Stop gracefully shuts down the mail manager, if running.
func Stop(ctx context.Context) {
	mu.Lock()
	defer mu.Unlock()

	if manager == nil {
		return
	}

	if err := manager.Stop(ctx); err != nil {
		log.Error().Err(err).Msg("mail: failed to stop manager")
	}
	manager = nil
	log.Info().Msg("mail: manager stopped")
}

// Restart stops (if running) and starts the mail manager to pick up new
// transport settings. Should only be called when the SMTP transport has
// actually changed; toggling recipients/flags does not require a restart as
// those are read live from config on each SendReport call.
func Restart(ctx context.Context) {
	Stop(ctx)
	if err := Start(ctx); err != nil {
		log.Error().Err(err).Msg("mail: failed to restart manager")
	}
}

// SendReport dispatches a refresh report according to the current
// notification settings. It returns quickly: the actual SMTP send happens in
// a background goroutine, and errors are logged.
//
// Dispatch rules:
//   - Notifications disabled → no-op.
//   - Success-severity reports when NotifyOnSuccess is false → no-op.
//   - Cooldown: at most one mail per severity per CooldownMinutes window.
//     A failure mail always flushes the success-cooldown, so operators see
//     failures immediately even after a recent success mail.
func SendReport(ctx context.Context, report Report) {
	cfg := config.Get()
	if !cfg.Notifications.Enabled {
		log.Debug().Msg("mail: notifications disabled, skipping report")
		return
	}

	if report.Severity == SeveritySuccess && !cfg.Notifications.NotifyOnSuccess {
		log.Debug().Msg("mail: success report suppressed by notify_on_success setting")
		return
	}

	if len(cfg.Notifications.Recipients) == 0 {
		log.Warn().Msg("mail: notifications enabled but no recipients configured, skipping report")
		return
	}

	if !allowBySeverity(report.Severity, cfg.Notifications.CooldownMinutes) {
		log.Debug().Msgf("mail: report of severity %s suppressed by cooldown", report.Severity)
		return
	}

	mu.Lock()
	m := manager
	mu.Unlock()
	if m == nil || !m.IsRunning() {
		log.Warn().Msg("mail: report requested but mail manager is not running")
		return
	}

	msg, err := buildMessage(cfg, report)
	if err != nil {
		log.Error().Err(err).Msg("mail: failed to build report message")
		return
	}

	go func() {
		// Decouple from the cron context so a task shutdown does not
		// immediately cancel the mail delivery; bound it to a sane
		// maximum nonetheless.
		sendCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := m.Send(sendCtx, msg); err != nil {
			log.Error().Err(err).Msg("mail: failed to send report")
			return
		}
		log.Info().Msgf("mail: sent %s report (%d updated, %d failed, %d unchanged)",
			report.Severity, report.UpdatedCount, report.FailedCount, report.UnchangedCount)
	}()
}

// allowBySeverity updates lastSentAt and returns true if the report passes
// the cooldown check for its severity. Failures always flush the
// success-cooldown so a prior success mail cannot delay a failure mail.
func allowBySeverity(sev Severity, cooldownMinutes int) bool {
	mu.Lock()
	defer mu.Unlock()

	if cooldownMinutes <= 0 {
		lastSentAt[sev] = time.Now()
		return true
	}

	cooldown := time.Duration(cooldownMinutes) * time.Minute
	last, ok := lastSentAt[sev]
	if ok && time.Since(last) < cooldown {
		return false
	}

	lastSentAt[sev] = time.Now()
	if sev == SeverityFailure {
		// Ensure a follow-up success mail after a failure has a fresh
		// cooldown window as well.
		delete(lastSentAt, SeveritySuccess)
	}
	return true
}

func buildMessage(cfg config.App, report Report) (*mail.Message, error) {
	prefix := strings.TrimSpace(cfg.Notifications.SubjectPrefix)

	var subject string
	switch {
	case report.AddressError != "" && report.FailedCount == 0:
		subject = fmt.Sprintf("%s refresh failed – address lookup error", prefix)
	case report.FailedCount > 0:
		subject = fmt.Sprintf("%s refresh: %d failed, %d updated", prefix, report.FailedCount, report.UpdatedCount)
	default:
		subject = fmt.Sprintf("%s refresh: %d updated", prefix, report.UpdatedCount)
	}

	text := renderText(report)

	builder := mail.NewMessage().
		To(cfg.Notifications.Recipients...).
		Subject(subject).
		TextBody(text).
		Template(reportTemplateName, report)
	if cfg.Mail.From != "" {
		builder = builder.From(cfg.Mail.From)
	}

	msg, err := builder.Build()
	if err != nil {
		return nil, apperror.NewError("failed to build report message").AddError(err)
	}
	return msg, nil
}

func renderText(r Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "HDNS DNS refresh report\n")
	fmt.Fprintf(&b, "=======================\n\n")
	if r.CurrentIPv4 != "" {
		fmt.Fprintf(&b, "IPv4:       %s\n", r.CurrentIPv4)
	}
	if r.CurrentIPv6 != "" {
		fmt.Fprintf(&b, "IPv6:       %s\n", r.CurrentIPv6)
	}
	if r.AddressError != "" {
		fmt.Fprintf(&b, "\nAddress lookup error:\n  %s\n", r.AddressError)
	}
	fmt.Fprintf(&b, "\nSummary:    %d updated, %d failed, %d unchanged (total %d)\n",
		r.UpdatedCount, r.FailedCount, r.UnchangedCount, len(r.Records))

	if len(r.Records) > 0 {
		fmt.Fprintf(&b, "\nRecords:\n")
		for _, rec := range r.Records {
			line := fmt.Sprintf("  [%s] %s.%s", strings.ToUpper(rec.Result), rec.Name, rec.Domain)
			if rec.Type != "" {
				line += " (" + rec.Type + ")"
			}
			if rec.Value != "" {
				line += fmt.Sprintf(" %s", emptyDash(rec.Value))
			}
			fmt.Fprintln(&b, line)
			if rec.Error != "" {
				fmt.Fprintf(&b, "      error: %s\n", rec.Error)
			}
		}
	}

	return b.String()
}

func emptyDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func statusColor(result string) string {
	switch result {
	case "updated":
		return "#0a7"
	case "failed":
		return "#c33"
	default:
		return "#666"
	}
}
