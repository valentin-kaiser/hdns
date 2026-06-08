package dns

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/flag"
	"github.com/valentin-kaiser/go-core/logging/log"
	"github.com/valentin-kaiser/hdns/pkg/config"
	"github.com/valentin-kaiser/hdns/pkg/database/schema"
)

// user implements the lego registration.User interface and represents the
// ACME account used to issue certificates.
type user struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *user) GetEmail() string                        { return u.Email }
func (u *user) GetRegistration() *registration.Resource { return u.Registration }
func (u *user) GetPrivateKey() crypto.PrivateKey        { return u.key }

// dir returns the directory used to persist ACME account material
// for the selected environment (staging or production).
func dir(staging bool) string {
	env := "production"
	if staging {
		env = "staging"
	}
	return filepath.Join(flag.Path, "acme", env)
}

// loadOrCreateUser loads the persisted ACME account key and registration
// for the given environment, generating a new account key if none exists.
func loadOrCreateUser(email string, staging bool) (*user, error) {
	dir := dir(staging)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, apperror.NewError("failed to create acme account directory").AddError(err)
	}

	keyPath := filepath.Join(dir, "account.key")
	regPath := filepath.Join(dir, "account.json")
	user := &user{Email: email}

	keyData, err := os.ReadFile(keyPath)
	switch {
	case err == nil:
		block, _ := pem.Decode(keyData)
		if block == nil {
			return nil, apperror.NewError("failed to decode acme account key")
		}
		key, perr := x509.ParseECPrivateKey(block.Bytes)
		if perr != nil {
			return nil, apperror.NewError("failed to parse acme account key").AddError(perr)
		}
		user.key = key

		regData, rerr := os.ReadFile(regPath)
		if rerr == nil {
			var reg registration.Resource
			if uerr := json.Unmarshal(regData, &reg); uerr == nil {
				user.Registration = &reg
			}
		}
		return user, nil
	case !os.IsNotExist(err):
		return nil, apperror.NewError("failed to read acme account key").AddError(err)
	}

	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, apperror.NewError("failed to generate acme account key").AddError(err)
	}
	der, err := x509.MarshalECPrivateKey(pk)
	if err != nil {
		return nil, apperror.NewError("failed to marshal acme account key").AddError(err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(keyPath, pemBytes, 0o600); err != nil {
		return nil, apperror.NewError("failed to persist acme account key").AddError(err)
	}
	user.key = pk
	return user, nil
}

// saveACMERegistration persists the ACME account registration resource.
func saveACMERegistration(user *user, staging bool) error {
	if user.Registration == nil {
		return nil
	}
	data, err := json.MarshalIndent(user.Registration, "", "  ")
	if err != nil {
		return apperror.NewError("failed to marshal acme registration").AddError(err)
	}
	regPath := filepath.Join(dir(staging), "account.json")
	if err := os.WriteFile(regPath, data, 0o600); err != nil {
		return apperror.NewError("failed to persist acme registration").AddError(err)
	}
	return nil
}

// newACMEClient builds a lego client for the configured account, registering
// the account with the ACME CA on first use.
func newACMEClient(email string, staging bool) (*lego.Client, error) {
	if strings.TrimSpace(email) == "" {
		return nil, apperror.NewError("acme account email is not configured")
	}

	user, err := loadOrCreateUser(email, staging)
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	cfg := lego.NewConfig(user)
	cfg.CADirURL = lego.LEDirectoryProduction
	if staging {
		cfg.CADirURL = lego.LEDirectoryStaging
	}
	cfg.Certificate.KeyType = certcrypto.RSA2048

	client, err := lego.NewClient(cfg)
	if err != nil {
		return nil, apperror.NewError("failed to create acme client").AddError(err)
	}

	if user.Registration == nil {
		reg, rerr := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if rerr != nil {
			return nil, apperror.NewError("failed to register acme account").AddError(rerr)
		}
		user.Registration = reg
		if serr := saveACMERegistration(user, staging); serr != nil {
			log.Warn().Err(serr).Msg("[ACME] failed to persist account registration")
		}
	}

	return client, nil
}

// provider implements the lego challenge.Provider interface by
// publishing ACME DNS-01 challenge tokens as TXT records via the Hetzner
// Cloud DNS API.
type provider struct {
	client *hcloud.Client
	zoneID int64
	zone   string

	mu     sync.Mutex
	values map[string][]string
}

func newProvider(client *hcloud.Client, zoneID int64, zone string) *provider {
	return &provider{
		client: client,
		zoneID: zoneID,
		zone:   zone,
		values: make(map[string][]string),
	}
}

// Present publishes the challenge token as a TXT record. Multiple challenges
// for the same FQDN (e.g. base domain + wildcard) are aggregated into a single
// RRSet.
func (p *provider) Present(domain, _, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	name := p.relativeName(info.FQDN)

	p.mu.Lock()
	p.values[name] = append(p.values[name], info.Value)
	vals := append([]string(nil), p.values[name]...)
	p.mu.Unlock()

	return upsertTXTRecord(context.Background(), p.client, p.zoneID, name, vals, 60)
}

// CleanUp removes the challenge TXT record.
func (p *provider) CleanUp(domain, _, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	name := p.relativeName(info.FQDN)

	p.mu.Lock()
	delete(p.values, name)
	p.mu.Unlock()

	return deleteTXTRecord(context.Background(), p.client, p.zoneID, name)
}

// relativeName converts an absolute challenge FQDN into a name relative to the
// zone, matching the convention used by the Hetzner Cloud DNS API ("@" for the
// zone apex).
func (p *provider) relativeName(fqdn string) string {
	fqdn = strings.TrimSuffix(fqdn, ".")
	zone := strings.TrimSuffix(p.zone, ".")
	switch {
	case fqdn == zone:
		return "@"
	case strings.HasSuffix(fqdn, "."+zone):
		return strings.TrimSuffix(fqdn, "."+zone)
	default:
		return fqdn
	}
}

// obtainCertificate runs the ACME DNS-01 flow for the given domains using the
// record's Hetzner credentials and zone.
func obtainCertificate(record *schema.Record, domains []string, zoneName string) (*certificate.Resource, error) {
	cfg := config.Get()

	client, err := newACMEClient(cfg.ACME.Email, cfg.ACME.Staging)
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	hclient, err := clientForRecord(record)
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	provider := newProvider(hclient, record.ZoneID, zoneName)

	opts := []dns01.ChallengeOption{}
	if servers := recursiveNameservers(); len(servers) > 0 {
		opts = append(opts, dns01.AddRecursiveNameservers(servers))
	}

	if err := client.Challenge.SetDNS01Provider(provider, opts...); err != nil {
		return nil, apperror.NewError("failed to configure dns-01 provider").AddError(err)
	}

	res, err := client.Certificate.Obtain(certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	})
	if err != nil {
		return nil, apperror.NewError("failed to obtain certificate").AddError(err)
	}

	return res, nil
}

// recursiveNameservers returns the resolvers used by lego to verify DNS-01
// challenge propagation before asking the CA to validate.
func recursiveNameservers() []string {
	servers := config.Get().DNSServers
	out := make([]string, 0, len(servers))
	for _, s := range servers {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}
