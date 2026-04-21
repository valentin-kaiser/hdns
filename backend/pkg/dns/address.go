package dns

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"

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
		addr, err := q.GetCurrentAddress(ctx)
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

		addr.Ipv4 = sql.NullString{String: ipv4, Valid: ipv4 != ""}
		addr.Ipv6 = sql.NullString{String: ipv6, Valid: ipv6 != ""}
		addr.Current = true

		_, err = q.CreateAddress(ctx, schema.CreateAddressParams{
			Ipv4:    addr.Ipv4,
			Ipv6:    addr.Ipv6,
			Current: addr.Current,
		})
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
		ipv4, err = resolveIPv4Address(r)
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
		ipv6, err = resolveIPv6Address(r)
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
	return true
}
