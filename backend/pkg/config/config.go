package config

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/robfig/cron"
	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/config"
	"github.com/valentin-kaiser/go-core/flag"
	"github.com/valentin-kaiser/go-core/logging/log"
	"github.com/valentin-kaiser/go-core/mail"
	"github.com/valentin-kaiser/go-core/security"
	"github.com/valentin-kaiser/hdns/pkg/proto/service"
)

// encryptionKeyFile is the filename used to persist the AES-256 encryption
// key on disk, relative to the application data directory.
const encryptionKeyFile = ".key"

var (
	mutex sync.RWMutex
	key   []byte
)

type App struct {
	LogLevel        int                  `usage:"(0 = debug, 1 = info, 2 = warn, 3 = error, 4 = fatal, 5 = panic)" json:"log_level"`
	WebPort         int16                `usage:"port to bind the web server to" json:"web_port"`
	CertificatePath string               `usage:"path to the TLS certificate file" json:"certificate_file"`
	KeyPath         string               `usage:"path to the TLS key file" json:"key_file"`
	RefreshCron     string               `usage:"cron expression to schedule data refresh tasks" json:"refresh_cron"`
	DNSServers      []string             `usage:"list of DNS servers to use for lookups" json:"dns_servers"`
	IPv4Resolvers   []string             `usage:"list of IPv4 resolvers to determine public IP address (supports http(s):// and dns:// URIs)" json:"ipv4_resolvers"`
	IPv6Resolvers   []string             `usage:"list of IPv6 resolvers to determine public IP address (supports http(s):// and dns:// URIs)" json:"ipv6_resolvers"`
	Database        string               `usage:"database connection DSN" json:"database"`
	Mail            mail.ClientConfig    `usage:"SMTP transport configuration (YAML only; not exposed via the web UI)" json:"mail"`
	Notifications   NotificationSettings `usage:"user-facing notification settings controlling when refresh reports are sent" json:"notifications"`
	ACME            ACMESettings         `usage:"ACME / Let's Encrypt settings for DNS-01 certificate issuance" json:"acme"`
}

// ACMESettings holds the configuration for issuing and renewing Let's Encrypt
// certificates via the DNS-01 challenge over the Hetzner Cloud DNS API.
type ACMESettings struct {
	// Enabled toggles whether hdns attempts to issue/renew certificates.
	Enabled bool `usage:"enable ACME certificate issuance and renewal" json:"enabled"`
	// Email is the account contact address registered with the ACME CA.
	Email string `usage:"account email address registered with the ACME CA" json:"email"`
	// Staging uses the Let's Encrypt staging environment when true.
	Staging bool `usage:"use the Let's Encrypt staging environment (untrusted certificates)" json:"staging"`
	// RenewBeforeDays is the number of days before expiry that a certificate
	// becomes eligible for renewal.
	RenewBeforeDays int `usage:"renew certificates this many days before they expire" json:"renew_before_days"`
	// RenewCron is the cron expression used to schedule renewal scans.
	RenewCron string `usage:"cron expression to schedule certificate renewal scans" json:"renew_cron"`
}

// NotificationSettings holds the user-facing notification behavior for DNS
// refresh reports. SMTP transport lives in App.Mail and is intentionally not
// part of this struct (and not exposed via the web UI).
type NotificationSettings struct {
	// Enabled toggles whether hdns attempts to send refresh reports at all.
	Enabled bool `usage:"enable sending of DNS refresh report emails" json:"enabled"`
	// NotifyOnSuccess also sends a report when at least one record was
	// actually updated during a successful refresh run (no failures).
	NotifyOnSuccess bool `usage:"also send a report when records were updated successfully" json:"notify_on_success"`
	// Recipients is the list of addresses that receive the reports.
	Recipients []string `usage:"recipient email addresses for refresh reports" json:"recipients"`
	// CooldownMinutes is the minimum number of minutes between two
	// dispatches of the same severity. 0 disables the cooldown.
	CooldownMinutes int `usage:"minimum minutes between report emails of the same severity (0 = no cooldown)" json:"cooldown_minutes"`
	// SubjectPrefix is prepended to the email subject.
	SubjectPrefix string `usage:"prefix prepended to the subject of refresh report emails" json:"subject_prefix"`
}

func Init() {
	defaultConfig := &App{
		LogLevel:        1,
		WebPort:         443,
		CertificatePath: filepath.Join(flag.Path, "certs/hdns.cert"),
		KeyPath:         filepath.Join(flag.Path, "certs/hdns.key"),
		RefreshCron:     "*/5 * * * *",
		DNSServers: []string{
			// Robot
			"hydrogen.ns.hetzner.com:53",
			"oxygen.ns.hetzner.com:53",
			"helium.ns.hetzner.de:53",
			// Konsole
			"ns3.second-ns.de:53",
			"ns1.your-server.de:53",
			"ns.second-ns.com:53",
			// Public
			"9.9.9.9:53",
			"1.1.1.1:53",
		},
		IPv4Resolvers: []string{
			"dns://resolver1.opendns.com/myip.opendns.com?type=A",
			"dns://ns1.google.com/o-o.myaddr.l.google.com?type=TXT",
			"dns://1.1.1.1/whoami.cloudflare?type=TXT&class=CH",
			"https://icanhazip.com/",
			"https://ident.me/",
			"https://api.ipy.ch",
		},
		IPv6Resolvers: []string{
			"dns://resolver1.opendns.com/myip.opendns.com?type=AAAA",
			"dns://ns1.google.com/o-o.myaddr.l.google.com?type=TXT",
			"dns://[2606:4700:4700::1111]/whoami.cloudflare?type=TXT&class=CH",
			"https://api6.ipify.org",
			"https://ipv6.icanhazip.com",
		},
		Database: "hdns:hdns@tcp(localhost:3306)/hdns?parseTime=true",
		Mail: mail.ClientConfig{
			Enabled: false,
		},
		Notifications: NotificationSettings{
			Enabled:         false,
			NotifyOnSuccess: false,
			Recipients:      []string{},
			CooldownMinutes: 60,
			SubjectPrefix:   "[HDNS]",
		},
		ACME: ACMESettings{
			Enabled:         false,
			Email:           "",
			Staging:         true,
			RenewBeforeDays: 30,
			RenewCron:       "0 3 * * *",
		},
	}

	err := config.Manager().WithName("hdns").Register(defaultConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to register configuration")
	}

	flag.Init()
	err = config.Read()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read configuration")
	}

	err = loadEncryptionKey()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load encryption key")
	}
}

// EncryptionKey returns the raw 32-byte AES-256 key used to encrypt
// Hetzner API tokens at rest. The key is loaded from or generated into
// the ".key" file inside the application data directory.
func EncryptionKey() []byte {
	mutex.RLock()
	defer mutex.RUnlock()
	keyCopy := make([]byte, len(key))
	copy(keyCopy, key)
	return keyCopy
}

// loadEncryptionKey reads the encryption key from disk, creating and
// persisting a new random key if the file does not yet exist.
func loadEncryptionKey() error {
	path := filepath.Join(flag.Path, encryptionKeyFile)

	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if len(data) != 32 {
			return apperror.NewError("encryption key file has unexpected size")
		}
		mutex.Lock()
		key = data
		mutex.Unlock()
		return nil
	case !os.IsNotExist(err):
		return apperror.NewError("failed to read encryption key file").AddError(err)
	}

	keyBytes, err := security.GetRandomBytes(32)
	if err != nil {
		return apperror.NewError("failed to generate encryption key").AddError(err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return apperror.NewError("failed to create encryption key directory").AddError(err)
	}

	if err := os.WriteFile(path, keyBytes, 0o600); err != nil {
		return apperror.NewError("failed to persist encryption key").AddError(err)
	}

	mutex.Lock()
	key = keyBytes
	mutex.Unlock()
	return nil
}

func Get() App {
	bc, ok := config.Get().(*App)
	if !ok {
		return App{}
	}

	if bc == nil {
		return App{}
	}

	return *bc
}

func Write(change *App) error {
	return apperror.Wrap(config.Write(change))
}

func OnChange(f func(o *App, n *App) error) {
	config.OnChange(func(o config.Config, n config.Config) error {
		if o == nil || n == nil {
			return apperror.NewError("the configuration provided is nil")
		}

		oc, ok := o.(*App)
		if !ok {
			return apperror.NewError("the configuration provided is not a BackendConfig")
		}

		nc, ok := n.(*App)
		if !ok {
			return apperror.NewError("the configuration provided is not a BackendConfig")
		}

		return f(oc, nc)
	})
}

func (c *App) Validate() error {
	if c.WebPort <= 0 {
		return apperror.NewError("web port must be greater than zero")
	}

	_, err := cron.ParseStandard(c.RefreshCron)
	if err != nil {
		return apperror.NewError("invalid refresh cron expression").AddError(err)
	}

	if c.Notifications.CooldownMinutes < 0 {
		return apperror.NewError("notifications.cooldown_minutes must be >= 0")
	}

	if c.Notifications.Enabled {
		if len(c.Notifications.Recipients) == 0 {
			return apperror.NewError("notifications.recipients must not be empty when notifications are enabled")
		}
		if !c.Mail.Enabled {
			return apperror.NewError("mail.client.enabled must be true when notifications are enabled")
		}
		if c.Mail.Host == "" {
			return apperror.NewError("mail.client.host must be set when notifications are enabled")
		}
		if c.Mail.From == "" {
			return apperror.NewError("mail.client.from must be set when notifications are enabled")
		}
		if err := c.Mail.Validate(); err != nil {
			return apperror.NewError("invalid mail configuration").AddError(err)
		}
	}

	if c.ACME.Enabled {
		if c.ACME.Email == "" {
			return apperror.NewError("acme.email must be set when acme is enabled")
		}
		if c.ACME.RenewBeforeDays <= 0 {
			return apperror.NewError("acme.renew_before_days must be greater than zero")
		}
		if _, err := cron.ParseStandard(c.ACME.RenewCron); err != nil {
			return apperror.NewError("invalid acme renew cron expression").AddError(err)
		}
	}

	return nil
}

func (c *App) ToProto() *service.Configuration {
	return &service.Configuration{
		LogLevel:                     int32(c.LogLevel),
		RefreshCron:                  c.RefreshCron,
		DnsServers:                   c.DNSServers,
		Ipv4Resolvers:                c.IPv4Resolvers,
		Ipv6Resolvers:                c.IPv6Resolvers,
		NotificationsEnabled:         c.Notifications.Enabled,
		NotificationsOnSuccess:       c.Notifications.NotifyOnSuccess,
		NotificationsRecipients:      c.Notifications.Recipients,
		NotificationsCooldownMinutes: int32(c.Notifications.CooldownMinutes),
		NotificationsSubjectPrefix:   c.Notifications.SubjectPrefix,
		AcmeEnabled:                  c.ACME.Enabled,
		AcmeEmail:                    c.ACME.Email,
		AcmeStaging:                  c.ACME.Staging,
		AcmeRenewBeforeDays:          int32(c.ACME.RenewBeforeDays),
	}
}

func (c *App) FromProto(pc *service.Configuration) *App {
	if pc == nil {
		return nil
	}
	c.LogLevel = int(pc.LogLevel)
	c.RefreshCron = pc.RefreshCron
	c.DNSServers = pc.DnsServers
	c.IPv4Resolvers = pc.Ipv4Resolvers
	c.IPv6Resolvers = pc.Ipv6Resolvers
	c.Notifications.Enabled = pc.NotificationsEnabled
	c.Notifications.NotifyOnSuccess = pc.NotificationsOnSuccess
	c.Notifications.Recipients = pc.NotificationsRecipients
	c.Notifications.CooldownMinutes = int(pc.NotificationsCooldownMinutes)
	c.Notifications.SubjectPrefix = pc.NotificationsSubjectPrefix
	c.ACME.Enabled = pc.AcmeEnabled
	c.ACME.Email = pc.AcmeEmail
	c.ACME.Staging = pc.AcmeStaging
	if pc.AcmeRenewBeforeDays > 0 {
		c.ACME.RenewBeforeDays = int(pc.AcmeRenewBeforeDays)
	}
	return c
}
