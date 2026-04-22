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
	LogLevel        int      `usage:"(0 = debug, 1 = info, 2 = warn, 3 = error, 4 = fatal, 5 = panic)" json:"log_level"`
	WebPort         int16    `usage:"port to bind the web server to" json:"web_port"`
	CertificatePath string   `usage:"path to the TLS certificate file" json:"certificate_file"`
	KeyPath         string   `usage:"path to the TLS key file" json:"key_file"`
	RefreshCron     string   `usage:"cron expression to schedule data refresh tasks" json:"refresh_cron"`
	DNSServers      []string `usage:"list of DNS servers to use for lookups" json:"dns_servers"`
	IPv4Resolvers   []string `usage:"list of IPv4 resolvers to determine public IP address (supports http(s):// and dns:// URIs)" json:"ipv4_resolvers"`
	IPv6Resolvers   []string `usage:"list of IPv6 resolvers to determine public IP address (supports http(s):// and dns:// URIs)" json:"ipv6_resolvers"`
	Database        string   `usage:"database connection DSN" json:"database"`
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
			"8.8.8.8:53",
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
	return key
}

// loadEncryptionKey reads the encryption key from disk, creating and
// persisting a new random key if the file does not yet exist.
func loadEncryptionKey() error {
	path := filepath.Join(flag.Path, encryptionKeyFile)

	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if len(data) != 32 {
			return apperror.NewError("encryption key file has unexpected size").AddError(err)
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

	return nil
}

func (c *App) ToProto() *service.Configuration {
	return &service.Configuration{
		LogLevel:      int32(c.LogLevel),
		RefreshCron:   c.RefreshCron,
		DnsServers:    c.DNSServers,
		Ipv4Resolvers: c.IPv4Resolvers,
		Ipv6Resolvers: c.IPv6Resolvers,
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
	return c
}
