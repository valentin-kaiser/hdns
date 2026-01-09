package config

import (
	"strings"

	"github.com/robfig/cron"
	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/config"
	"github.com/valentin-kaiser/go-core/flag"
	"github.com/valentin-kaiser/go-core/logging/log"
)

type App struct {
	Service  Service  `usage:"service settings" json:"service"`
	Database Database `usage:"database connection settings" json:"database"`
}

type Service struct {
	LogLevel        int      `usage:"(0 = debug, 1 = info, 2 = warn, 3 = error, 4 = fatal, 5 = panic)" json:"log_level"`
	WebPort         int16    `usage:"port to bind the web server to" json:"web_port"`
	CertificateFile string   `usage:"path to the TLS certificate file" json:"certificate_file"`
	KeyFile         string   `usage:"path to the TLS key file" json:"key_file"`
	RefreshCron     string   `usage:"cron expression to schedule data refresh tasks" json:"refresh_cron"`
	DNSServers      []string `usage:"list of DNS servers to use for lookups" json:"dns_servers"`
}

type Database struct {
	Driver   string
	Host     string
	Port     uint16
	Username string
	Password string
	Name     string
}

func Init() {
	defaultConfig := &App{
		Service: Service{
			LogLevel:    1,
			WebPort:     9100,
			RefreshCron: "@every 5m",
			DNSServers:  []string{"9.9.9.9:53", "1.1.1.1:53", "8.8.8.8:53"},
		},
		Database: Database{
			Driver:   "sqlite",
			Host:     "localhost",
			Port:     3306,
			Username: "hdns",
			Password: "hdns",
			Name:     "hdns",
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

	return nil
}

func (c *Service) Validate() error {
	if c.WebPort <= 0 {
		return apperror.NewError("web port must be greater than zero")
	}

	_, err := cron.ParseStandard(c.RefreshCron)
	if err != nil {
		return apperror.NewError("invalid refresh cron expression").AddError(err)
	}

	return nil
}

func (c *Database) Validate() error {
	if strings.TrimSpace(c.Driver) == "" {
		return apperror.NewError("database driver is required")
	}

	if c.Driver != "sqlite" && c.Driver != "mysql" {
		return apperror.NewErrorf("unsupported database driver: %s", c.Driver)
	}

	if strings.TrimSpace(c.Host) == "" {
		return apperror.NewError("database host is required")
	}

	if c.Port == 0 {
		return apperror.NewError("database port is required")
	}

	if strings.TrimSpace(c.Username) == "" {
		return apperror.NewError("database username is required")
	}

	if strings.TrimSpace(c.Password) == "" {
		return apperror.NewError("database password is required")
	}

	if strings.TrimSpace(c.Name) == "" {
		return apperror.NewError("database name is required")
	}

	return nil
}
