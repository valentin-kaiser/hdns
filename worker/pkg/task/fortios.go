package task

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/valentin-kaiser/hdns-worker/pkg/config"
)

type fortiOSUploadRequest struct {
	Type        string `json:"type"`
	CertName    string `json:"certname"`
	Scope       string `json:"scope,omitempty"`
	FileContent string `json:"file_content"`
}

type fortiOSAPIResponse struct {
	Status     string `json:"status"`
	HTTPStatus int    `json:"http_status"`
	Error      string `json:"error"`
	Message    string `json:"message"`
}

// uploadFortiOS imports the PKCS#12 certificate from the webhook payload into
// a FortiGate via the monitor API endpoint.
func uploadFortiOS(ctx context.Context, action config.ActionConfig, payload Payload) error {
	if action.FortiOS == nil {
		return fmt.Errorf("fortios_upload: fortios config is not set")
	}
	if action.FortiOS.DryRun {
		return verifyFortiOSConnection(ctx, action.FortiOS)
	}

	encoded, err := payloadPKCS12Base64(payload)
	if err != nil {
		return err
	}

	reqBody := fortiOSUploadRequest{
		Type:        "pkcs12",
		CertName:    action.FortiOS.CertName,
		Scope:       action.FortiOS.Scope,
		FileContent: encoded,
	}
	_, err = doFortiOSRequest(ctx, action.FortiOS, http.MethodPost, "/api/v2/monitor/vpn-certificate/local/import", reqBody)
	if err != nil {
		return fmt.Errorf("fortios_upload: %w", err)
	}

	return nil
}

// replaceFortiOSProfileCert updates one or more configured CMDB profile fields
// so they reference the imported certificate name.
func replaceFortiOSProfileCert(ctx context.Context, action config.ActionConfig) error {
	if action.FortiOS == nil {
		return fmt.Errorf("fortios_profile_cert_replace: fortios config is not set")
	}
	if action.FortiOS.DryRun {
		return verifyFortiOSConnection(ctx, action.FortiOS)
	}

	for i, update := range action.ProfileUpdates {
		cmdbPath := buildFortiOSCMDBPath(update.Path, update.MKey)
		body := map[string]string{update.Field: action.FortiOS.CertName}
		if _, err := doFortiOSRequest(ctx, action.FortiOS, update.Method, cmdbPath, body); err != nil {
			return fmt.Errorf("fortios_profile_cert_replace: update[%d] path=%q field=%q: %w", i, cmdbPath, update.Field, err)
		}
	}

	return nil
}

// updateFortiOSAdminServerCert sets /api/v2/cmdb/system/global admin-server-cert
// to the configured certificate name.
func updateFortiOSAdminServerCert(ctx context.Context, action config.ActionConfig) error {
	if action.FortiOS == nil {
		return fmt.Errorf("fortios_admin_server_cert_update: fortios config is not set")
	}
	if action.FortiOS.DryRun {
		return verifyFortiOSConnection(ctx, action.FortiOS)
	}

	body := map[string]string{"admin-server-cert": action.FortiOS.CertName}
	if _, err := doFortiOSRequest(ctx, action.FortiOS, http.MethodPut, "/api/v2/cmdb/system/global", body); err != nil {
		return fmt.Errorf("fortios_admin_server_cert_update: %w", err)
	}

	return nil
}

func payloadPKCS12Base64(payload Payload) (string, error) {
	pkcs12Base64 := compactBase64(payload.PKCS12Base64)
	if pkcs12Base64 != "" {
		return pkcs12Base64, nil
	}

	raw := strings.TrimSpace(payload.PKCS12)
	if raw == "" {
		return "", fmt.Errorf("fortios_upload: payload contains no pkcs12 data")
	}

	candidate := compactBase64(raw)
	if _, err := base64.StdEncoding.DecodeString(candidate); err == nil {
		return candidate, nil
	}

	return base64.StdEncoding.EncodeToString([]byte(raw)), nil
}

func verifyFortiOSConnection(ctx context.Context, cfg *config.FortiOSConfig) error {
	_, err := doFortiOSRequest(ctx, cfg, http.MethodGet, "/api/v2/cmdb/system/global", nil)
	if err != nil {
		return fmt.Errorf("fortios dry_run verify failed: %w", err)
	}
	return nil
}

func doFortiOSRequest(ctx context.Context, cfg *config.FortiOSConfig, method string, apiPath string, body any) ([]byte, error) {
	endpoint, err := fortiOSEndpoint(cfg, apiPath)
	if err != nil {
		return nil, err
	}

	var reqBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)

	resp, err := fortiOSHTTPClient(cfg).Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if readErr != nil {
		return nil, fmt.Errorf("read response: %w", readErr)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncateText(string(respBytes), 400))
	}

	var apiResp fortiOSAPIResponse
	if len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, &apiResp); err == nil {
			if apiResp.Status != "" && apiResp.Status != "success" {
				return nil, fmt.Errorf("API status %q: %s", apiResp.Status, truncateText(string(respBytes), 400))
			}
			if apiResp.HTTPStatus != 0 && apiResp.HTTPStatus != http.StatusOK {
				return nil, fmt.Errorf("API http_status=%d: %s", apiResp.HTTPStatus, truncateText(string(respBytes), 400))
			}
		}
	}

	return respBytes, nil
}

func fortiOSHTTPClient(cfg *config.FortiOSConfig) *http.Client {
	client := &http.Client{}
	if cfg.TLSInsecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // explicit per task config
		}
	}
	return client
}

func fortiOSEndpoint(cfg *config.FortiOSConfig, apiPath string) (string, error) {
	base := strings.TrimSpace(cfg.Host)
	if base == "" {
		return "", fmt.Errorf("fortios_upload: fortios.host is empty")
	}
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "https://" + base
	}
	base = strings.TrimRight(base, "/")

	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("fortios_upload: invalid fortios.host: %w", err)
	}

	u.Path = strings.TrimRight(u.Path, "/") + ensureLeadingSlash(apiPath)
	q := u.Query()
	q.Set("access_token", cfg.AccessToken)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func buildFortiOSCMDBPath(rawPath string, mkey string) string {
	clean := strings.Trim(strings.TrimSpace(rawPath), "/")
	apiPath := "/api/v2/cmdb/" + clean
	if strings.TrimSpace(mkey) != "" {
		apiPath = path.Join(apiPath, url.PathEscape(strings.TrimSpace(mkey)))
	}
	return apiPath
}

func ensureLeadingSlash(in string) string {
	if strings.HasPrefix(in, "/") {
		return in
	}
	return "/" + in
}

func compactBase64(in string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\n', '\r', '\t':
			return -1
		default:
			return r
		}
	}, in)
}

func truncateText(in string, max int) string {
	if len(in) <= max {
		return in
	}
	return in[:max] + "..."
}
