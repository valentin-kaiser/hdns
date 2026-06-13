package dns

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
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
	c := config.Get()

	ipv4, err4 := resolveWithConsensus(
		c.IPv4Resolvers,
		false,
		c.IPv4ResolverAgreementThreshold,
		c.IPv4ResolverMinResponses,
		"ipv4",
	)

	ipv6, err6 := resolveWithConsensus(
		c.IPv6Resolvers,
		true,
		c.IPv6ResolverAgreementThreshold,
		c.IPv6ResolverMinResponses,
		"ipv6",
	)

	if ipv4 == "" && ipv6 == "" {
		err := apperror.NewError("failed to resolve public IP address using all configured resolvers")
		if err4 != nil {
			err = err.AddError(err4)
		}
		if err6 != nil {
			err = err.AddError(err6)
		}
		return "", "", err
	}

	return ipv4, ipv6, nil
}

func resolveWithConsensus(resolvers []string, wantIPv6 bool, threshold float64, minResponses int, family string) (string, error) {
	if len(resolvers) == 0 {
		return "", apperror.NewErrorf("no %s resolvers configured", family)
	}

	counts := map[string]int{}
	firstSeen := map[string]int{}
	successful := 0

	for _, r := range resolvers {
		addr, err := resolveEntry(r, wantIPv6)
		if err != nil {
			log.Warn().Err(err).Field("family", family).Msgf("resolver %s failed", r)
			continue
		}

		if _, ok := firstSeen[addr]; !ok {
			firstSeen[addr] = successful
		}
		counts[addr]++
		successful++

		log.Debug().
			Field("family", family).
			Field("resolver", r).
			Field("candidate", addr).
			Msg("resolved candidate address")
	}

	if successful == 0 {
		return "", apperror.NewErrorf("failed to resolve %s address using all configured resolvers", family)
	}

	selected, winnerCount := pickConsensusCandidate(counts, firstSeen)
	agreement := float64(winnerCount) / float64(successful)

	logEvt := log.Info()
	if successful < minResponses || agreement < threshold {
		logEvt = log.Warn()
	}

	logEvt.
		Field("family", family).
		Field("selected", selected).
		Field("winner_count", winnerCount).
		Field("successful_responses", successful).
		Field("agreement_ratio", agreement).
		Field("threshold", threshold).
		Field("min_responses", minResponses).
		Field("distribution", formatCandidateDistribution(counts)).
		Msg("resolved public IP using resolver consensus")

	if successful < minResponses || agreement < threshold {
		log.Warn().
			Field("family", family).
			Field("selected", selected).
			Field("agreement_ratio", agreement).
			Field("threshold", threshold).
			Field("successful_responses", successful).
			Field("min_responses", minResponses).
			Msg("resolver consensus below trust threshold, using best-effort candidate")
	}

	return selected, nil
}

func pickConsensusCandidate(counts map[string]int, firstSeen map[string]int) (string, int) {
	bestAddr := ""
	bestCount := -1
	bestFirstSeen := 0

	for addr, count := range counts {
		if count > bestCount {
			bestAddr = addr
			bestCount = count
			bestFirstSeen = firstSeen[addr]
			continue
		}

		if count == bestCount && firstSeen[addr] < bestFirstSeen {
			bestAddr = addr
			bestFirstSeen = firstSeen[addr]
		}
	}

	return bestAddr, bestCount
}

func formatCandidateDistribution(counts map[string]int) string {
	parts := make([]string, 0, len(counts))
	keys := make([]string, 0, len(counts))
	for addr := range counts {
		keys = append(keys, addr)
	}
	sort.Strings(keys)

	for _, addr := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", addr, counts[addr]))
	}

	return strings.Join(parts, ",")
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
