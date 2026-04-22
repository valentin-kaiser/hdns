package dns

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	miekgdns "github.com/miekg/dns"
	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/logging/log"
	"github.com/valentin-kaiser/go-core/version"
	"github.com/valentin-kaiser/hdns/pkg/config"
	"github.com/valentin-kaiser/hdns/pkg/database"
	"github.com/valentin-kaiser/hdns/pkg/database/schema"
)

func UpdateAddress(ctx context.Context) (*schema.Address, error) {
	ipv4, ipv6, err := resolve()
	if err != nil {
		return nil, apperror.Wrap(err)
	}

	var addr *schema.Address
	err = database.HDNS().Query(func(q *schema.Queries) error {
		addr, err = q.GetCurrentAddress(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return apperror.Wrap(err)
		}

		if addr.Ipv4.String == ipv4 && addr.Ipv6.String == ipv6 {
			log.Info().Field("ipv4", ipv4).Field("ipv6", ipv6).Msg("public IP address is already up-to-date")
			return nil
		}

		err = q.ResetCurrentAddresses(ctx)
		if err != nil {
			return apperror.Wrap(err)
		}

		_, err = q.CreateAddress(ctx, schema.CreateAddressParams{
			Ipv4:    sql.NullString{String: ipv4, Valid: ipv4 != ""},
			Ipv6:    sql.NullString{String: ipv6, Valid: ipv6 != ""},
			Current: true,
		})
		if err != nil {
			return apperror.Wrap(err)
		}

		addr, err = q.GetCurrentAddress(ctx)
		if err != nil {
			return apperror.Wrap(err)
		}

		return nil
	})
	if err != nil {
		return nil, apperror.NewError("failed to save public IP address to database").AddError(err)
	}
	return addr, nil
}

func resolve() (string, string, error) {
	var ipv4 string
	for _, r := range config.Get().IPv4Resolvers {
		var err error
		ipv4, err = resolveEntry(r, false)
		if err != nil {
			log.Warn().Err(err).Msgf("resolver %s failed", r)
			continue
		}
		log.Info().Field("ipv4", ipv4).Field("resolver", r).Msg("resolved public IP")
		break
	}

	var ipv6 string
	for _, r := range config.Get().IPv6Resolvers {
		var err error
		ipv6, err = resolveEntry(r, true)
		if err != nil {
			log.Warn().Err(err).Msgf("resolver %s failed", r)
			continue
		}
		log.Info().Field("ipv6", ipv6).Field("resolver", r).Msg("resolved public IPv6")
		break
	}

	if ipv4 == "" && ipv6 == "" {
		return "", "", apperror.NewError("failed to resolve public IP address using all configured resolvers")
	}

	return ipv4, ipv6, nil
}

func resolveEntry(entry string, wantIPv6 bool) (string, error) {
	u, err := url.Parse(entry)
	if err != nil {
		return "", apperror.NewErrorf("invalid resolver entry %s", entry).AddError(err)
	}

	switch strings.ToLower(u.Scheme) {
	case "", "http", "https":
		if wantIPv6 {
			return resolveIPv6Address(entry)
		}
		return resolveIPv4Address(entry)
	case "dns":
		return resolveViaDNS(u, wantIPv6)
	default:
		return "", apperror.NewErrorf("unsupported resolver scheme %q in %s", u.Scheme, entry)
	}
}

func resolveIPv4Address(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", apperror.NewErrorf("failed to create request for %s", url).AddError(err)
	}
	req.Header.Set("User-Agent", "hdns/"+version.GitTag)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", apperror.NewErrorf("failed to get public IP from %s", url).AddError(err)
	}
	defer apperror.Catch(resp.Body.Close, "failed to close response body")

	bytes, err := io.ReadAll(io.LimitReader(resp.Body, 15))
	if err != nil {
		return "", apperror.NewErrorf("failed to read response from %s", url).AddError(err)
	}

	addr := strings.TrimSpace(string(bytes))
	if !ValidateIpv4Address(addr) {
		return "", apperror.NewErrorf("invalid IP address %s from %s", addr, url)
	}
	return addr, nil
}

func resolveIPv6Address(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", apperror.NewErrorf("failed to create request for %s", url).AddError(err)
	}
	req.Header.Set("User-Agent", "hdns/"+version.GitTag)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", apperror.NewErrorf("failed to get public IPv6 from %s", url).AddError(err)
	}
	defer apperror.Catch(resp.Body.Close, "failed to close response body")

	bytes, err := io.ReadAll(io.LimitReader(resp.Body, 45))
	if err != nil {
		return "", apperror.NewErrorf("failed to read response from %s", url).AddError(err)
	}

	addr := strings.TrimSpace(string(bytes))
	if !ValidateIpv6Address(addr) {
		return "", apperror.NewErrorf("invalid IPv6 address %s from %s", addr, url)
	}
	return addr, nil
}

func resolveViaDNS(u *url.URL, wantIPv6 bool) (string, error) {
	server := u.Host
	if server == "" {
		return "", apperror.NewErrorf("dns resolver %s missing server host", u.String())
	}
	if _, _, err := net.SplitHostPort(server); err != nil {
		server = net.JoinHostPort(u.Hostname(), "53")
	}

	name := strings.TrimPrefix(u.Path, "/")
	if name == "" {
		return "", apperror.NewErrorf("dns resolver %s missing query name", u.String())
	}
	name = miekgdns.Fqdn(name)

	q := u.Query()
	qtypeStr := strings.ToUpper(q.Get("type"))
	if qtypeStr == "" {
		qtypeStr = "A"
		if wantIPv6 {
			qtypeStr = "AAAA"
		}
	}
	qtype, ok := miekgdns.StringToType[qtypeStr]
	if !ok {
		return "", apperror.NewErrorf("unsupported dns query type %q in %s", qtypeStr, u.String())
	}

	qclassStr := strings.ToUpper(q.Get("class"))
	if qclassStr == "" {
		qclassStr = "IN"
	}
	qclass, ok := miekgdns.StringToClass[qclassStr]
	if !ok {
		return "", apperror.NewErrorf("unsupported dns query class %q in %s", qclassStr, u.String())
	}

	msg := new(miekgdns.Msg)
	msg.SetQuestion(name, qtype)
	msg.Question[0].Qclass = qclass
	msg.RecursionDesired = true

	client := &miekgdns.Client{Net: "udp", Timeout: 5 * time.Second}
	resp, _, err := client.Exchange(msg, server)
	if err != nil {
		return "", apperror.NewErrorf("dns query to %s failed", server).AddError(err)
	}
	if resp.Truncated {
		client.Net = "tcp"
		resp, _, err = client.Exchange(msg, server)
		if err != nil {
			return "", apperror.NewErrorf("dns tcp query to %s failed", server).AddError(err)
		}
	}
	if resp.Rcode != miekgdns.RcodeSuccess {
		return "", apperror.NewErrorf("dns query to %s returned rcode %s", server, miekgdns.RcodeToString[resp.Rcode])
	}

	var addr string
	for _, rr := range resp.Answer {
		switch r := rr.(type) {
		case *miekgdns.A:
			if qtype == miekgdns.TypeA {
				addr = r.A.String()
			}
		case *miekgdns.AAAA:
			if qtype == miekgdns.TypeAAAA {
				addr = r.AAAA.String()
			}
		case *miekgdns.TXT:
			if qtype == miekgdns.TypeTXT && len(r.Txt) > 0 {
				addr = strings.Trim(strings.TrimSpace(strings.Join(r.Txt, "")), "\"")
			}
		}
		if addr != "" {
			break
		}
	}

	if addr == "" {
		return "", apperror.NewErrorf("dns query to %s returned no matching records", server)
	}

	if wantIPv6 {
		if !ValidateIpv6Address(addr) {
			return "", apperror.NewErrorf("invalid IPv6 address %s from dns %s", addr, u.String())
		}
	} else {
		if !ValidateIpv4Address(addr) {
			return "", apperror.NewErrorf("invalid IPv4 address %s from dns %s", addr, u.String())
		}
	}
	return addr, nil
}

func ValidateIpv4Address(ip string) bool {
	addr := net.ParseIP(ip)
	if addr == nil {
		return false
	}
	if addr.IsUnspecified() {
		return false
	}
	if addr.IsPrivate() {
		return false
	}
	if addr.IsLoopback() {
		return false
	}
	if addr.IsMulticast() {
		return false
	}
	if addr.To4() == nil {
		return false
	}
	return true
}

func ValidateIpv6Address(ip string) bool {
	addr := net.ParseIP(ip)
	if addr == nil {
		return false
	}
	if addr.IsUnspecified() {
		return false
	}
	if addr.IsLoopback() {
		return false
	}
	if addr.IsMulticast() {
		return false
	}
	if addr.To16() == nil {
		return false
	}
	if addr.To4() != nil {
		return false
	}
	return true
}
