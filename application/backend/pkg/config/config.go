package config

import (
	"path/filepath"
	"strings"

	"github.com/robfig/cron"
	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/config"
	"github.com/valentin-kaiser/go-core/flag"
	"github.com/valentin-kaiser/go-core/logging/log"
	"github.com/valentin-kaiser/hdns/pkg/proto/service"
)

type App struct {
	Service  Service  `usage:"service settings" json:"service"`
	Database Database `usage:"database connection settings" json:"database"`
}

type Service struct {
	LogLevel        int      `usage:"(0 = debug, 1 = info, 2 = warn, 3 = error, 4 = fatal, 5 = panic)" json:"log_level"`
	WebPort         int16    `usage:"port to bind the web server to" json:"web_port"`
	CertificatePath string   `usage:"path to the TLS certificate file" json:"certificate_file"`
	KeyPath         string   `usage:"path to the TLS key file" json:"key_file"`
	RefreshCron     string   `usage:"cron expression to schedule data refresh tasks" json:"refresh_cron"`
	DNSServers      []string `usage:"list of DNS servers to use for lookups" json:"dns_servers"`
	IPv4Resolvers   []string `usage:"list of IPv4 resolvers to determine public IP address" json:"ipv4_resolvers"`
	IPv6Resolvers   []string `usage:"list of IPv6 resolvers to determine public IP address" json:"ipv6_resolvers"`
}

type Database struct {
	Host     string
	Port     uint16
	Username string
	Password string
	Name     string
}

func Init() {
	defaultConfig := &App{
		Service: Service{
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
		},
		Database: Database{
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

func (c *App) ToProto() *service.Configuration {
	return &service.Configuration{
		Service: &service.Service{
			LogLevel:        int32(c.Service.LogLevel),
			WebPort:         int32(c.Service.WebPort),
			CertificatePath: c.Service.CertificatePath,
			KeyPath:         c.Service.KeyPath,
			RefreshCron:     c.Service.RefreshCron,
			DnsServers:      c.Service.DNSServers,
		},
		Database: &service.Database{
			Host:     c.Database.Host,
			Port:     uint32(c.Database.Port),
			Username: c.Database.Username,
			Password: c.Database.Password,
			Name:     c.Database.Name,
		},
	}
}

func (c *App) FromProto(pc *service.Configuration) *App {
	if pc == nil {
		return nil
	}
	if pc.Service == nil || pc.Database == nil {
		return nil
	}

	c.Service.LogLevel = int(pc.Service.LogLevel)
	c.Service.WebPort = int16(pc.Service.WebPort)
	c.Service.CertificatePath = pc.Service.CertificatePath
	c.Service.KeyPath = pc.Service.KeyPath
	c.Service.RefreshCron = pc.Service.RefreshCron
	c.Service.DNSServers = pc.Service.DnsServers
	c.Database.Host = pc.Database.Host
	c.Database.Port = uint16(pc.Database.Port)
	c.Database.Username = pc.Database.Username
	c.Database.Password = pc.Database.Password
	c.Database.Name = pc.Database.Name
	return c
}
