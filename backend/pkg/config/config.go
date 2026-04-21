package config

import (
	"path/filepath"

	"github.com/robfig/cron"
	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/config"
	"github.com/valentin-kaiser/go-core/database"
	"github.com/valentin-kaiser/go-core/flag"
	"github.com/valentin-kaiser/go-core/logging/log"
	"github.com/valentin-kaiser/hdns/pkg/proto/service"
)

type App struct {
	LogLevel        int             `usage:"(0 = debug, 1 = info, 2 = warn, 3 = error, 4 = fatal, 5 = panic)" json:"log_level"`
	WebPort         int16           `usage:"port to bind the web server to" json:"web_port"`
	CertificatePath string          `usage:"path to the TLS certificate file" json:"certificate_file"`
	KeyPath         string          `usage:"path to the TLS key file" json:"key_file"`
	RefreshCron     string          `usage:"cron expression to schedule data refresh tasks" json:"refresh_cron"`
	DNSServers      []string        `usage:"list of DNS servers to use for lookups" json:"dns_servers"`
	IPv4Resolvers   []string        `usage:"list of IPv4 resolvers to determine public IP address" json:"ipv4_resolvers"`
	IPv6Resolvers   []string        `usage:"list of IPv6 resolvers to determine public IP address" json:"ipv6_resolvers"`
	DatabaseDriver  database.Driver `usage:"database driver to use (e.g. sqlite, postgres, mysql)" json:"database_driver"`
	Database        string          `usage:"database connection DSN" json:"database"`
}

func Init() {
	defaultConfig := &App{
		LogLevel:        1,
		WebPort:         9100,
		CertificatePath: filepath.Join(flag.Path, "certs/hdns.cert"),
		KeyPath:         filepath.Join(flag.Path, "certs/hdns.key"),
		RefreshCron:     "0 */5 * * * *",
		DNSServers: []string{
			"9.9.9.9:53",
			"1.1.1.1:53",
			"8.8.8.8:53",
		},
		IPv4Resolvers: []string{
			"https://nms.intellitrend.de",
			"https://api.ipify.org",
			"https://api.my-ip.io/ip",
			"https://api.ipy.ch",
			"https://ident.me/",
			"https://ifconfig.me/ip",
			"https://icanhazip.com/",
		},
		IPv6Resolvers: []string{
			"https://api6.ipify.org",
			"https://ipv6.icanhazip.com",
		},
		Database: "file:hdns.db?cache=shared&_pragma=foreign_keys(1)",
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

	return nil
}

func (c *App) ToProto() *service.Configuration {
	return &service.Configuration{
		Service: &service.Service{
			LogLevel:        int32(c.LogLevel),
			WebPort:         int32(c.WebPort),
			CertificatePath: c.CertificatePath,
			KeyPath:         c.KeyPath,
			RefreshCron:     c.RefreshCron,
			DnsServers:      c.DNSServers,
		},
	}
}

func (c *App) FromProto(pc *service.Configuration) *App {
	if pc == nil {
		return nil
	}
	if pc.Service == nil {
		return nil
	}

	c.LogLevel = int(pc.Service.LogLevel)
	c.WebPort = int16(pc.Service.WebPort)
	c.CertificatePath = pc.Service.CertificatePath
	c.KeyPath = pc.Service.KeyPath
	c.RefreshCron = pc.Service.RefreshCron
	c.DNSServers = pc.Service.DnsServers
	c.DatabaseDriver = database.Driver(pc.Driver)
	c.Database = pc.Database
	return c
}
