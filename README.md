<p align="center">
  <img src="frontend/src/assets/hdns.png" width="365">
</p>

# HDNS - Hetzner Dynamic DNS Management

A modern web-based Dynamic DNS management solution specifically designed for Hetzner DNS services.
HDNS provides an intuitive interface for managing DNS records, automatically updating them with your current IP address, and **issuing trusted TLS certificates from Let's Encrypt** — all without exposing any additional ports.

Built on top of [Hetzner's Cloud API v2](https://pkg.go.dev/github.com/hetznercloud/hcloud-go/v2/hcloud), HDNS answers ACME DNS-01 challenges directly through the Hetzner DNS API, so you get fully automated certificate lifecycle management — issuance, renewal, and delivery to your services — alongside your existing Dynamic DNS workflows.

## Installation

```bash
docker pull ghcr.io/valentin-kaiser/hdns:latest
docker run -p 443:443 ghcr.io/valentin-kaiser/hdns:latest
```


```bash
# Clone the repository
git clone https://github.com/valentin-kaiser/hdns.git
cd hdns

# Build and run with Docker
docker build --tag hdns .
docker run -p 443:443 hdns
```

## Configuration

### Environment Variables

- `HDNS_LOG_LEVEL`: Log level (0 = debug, 1 = info, 2 = warn, 3 = error, 4 = fatal, 5 = panic)
- `HDNS_WEB_PORT`: Port to bind the web server to
- `HDNS_CERTIFICATE_PATH`: Path to the TLS certificate file
- `HDNS_KEY_PATH`: Path to the TLS key file
- `HDNS_REFRESH_CRON`: Cron expression to schedule data refresh tasks
- `HDNS_DNS_SERVERS`: Comma-separated list of DNS servers to use for lookups
- `HDNS_IPV4_RESOLVERS`: Comma-separated list of IPv4 resolvers to determine public IP address. Each entry is a URI; `http(s)://…` entries perform an HTTP GET and expect the plain IP in the body, while `dns://<server>[:port]/<query-name>?type=A|AAAA|TXT[&class=IN|CH]` entries perform a DNS query (e.g. `dns://resolver1.opendns.com/myip.opendns.com?type=A`, `dns://1.1.1.1/whoami.cloudflare?type=TXT&class=CH`).
- `HDNS_IPV6_RESOLVERS`: Comma-separated list of IPv6 resolvers to determine public IP address. Same URI format as `HDNS_IPV4_RESOLVERS`; for DNS entries the default query type is `AAAA`.
- `HDNS_DATABASE`: Database connection DSN
- `HDNS_ACME_ENABLED`: Enable ACME certificate issuance and renewal via Let's Encrypt
- `HDNS_ACME_EMAIL`: Account email address registered with the ACME CA
- `HDNS_ACME_STAGING`: Use the Let's Encrypt staging environment (issues untrusted certificates; default: `true`)
- `HDNS_ACME_RENEW_BEFORE_DAYS`: Renew certificates this many days before they expire (default: `30`)
- `HDNS_ACME_RENEW_CRON`: Cron expression for the certificate renewal scan (default: `0 3 * * *`)

### File-Based Configuration

```yaml
loglevel: 1
webport: 443
certificatepath: "/path/to/cert.pem"
keypath: "/path/to/key.pem"
refreshcron: "0 * * * *"
dnsservers:
- hydrogen.ns.hetzner.com:53
- oxygen.ns.hetzner.com:53
- helium.ns.hetzner.de:53
- ns3.second-ns.de:53
- ns1.your-server.de:53
- ns.second-ns.com:53
- 9.9.9.9:53
- 1.1.1.1:53
- 8.8.8.8:53
ipv4resolvers:
- dns://resolver1.opendns.com/myip.opendns.com?type=A
- dns://ns1.google.com/o-o.myaddr.l.google.com?type=TXT
- dns://1.1.1.1/whoami.cloudflare?type=TXT&class=CH
- https://icanhazip.com/
- https://ident.me/
- https://api.ipy.ch
ipv6resolvers:
- dns://resolver1.opendns.com/myip.opendns.com?type=AAAA
- dns://ns1.google.com/o-o.myaddr.l.google.com?type=TXT
- dns://[2606:4700:4700::1111]/whoami.cloudflare?type=TXT&class=CH
- https://api6.ipify.org
- https://ipv6.icanhazip.com
database: "hdns:hdns@tcp(localhost:3306)/hdns?parseTime=true"
acme:
  enabled: false
  email: "you@example.com"
  staging: true
  renewbeforedays: 30
  renewcron: "0 3 * * *"
```

## Let's Encrypt Certificates (ACME)

HDNS can automatically issue and renew TLS certificates from Let's Encrypt using the **DNS-01 challenge** over the Hetzner Cloud DNS API.  No additional ports or publicly reachable HTTP endpoints are required — the challenge is answered entirely via DNS TXT records.

### How it works

1. A DNS record is configured with the purpose set to **Certificate** (or **Both** for DDNS + certificate).
2. Issuance is triggered manually from the UI or automatically by the renewal scheduler.
3. HDNS creates a temporary `_acme-challenge` TXT record in the Hetzner zone, waits for propagation, and completes the DNS-01 challenge.
4. The signed certificate and private key are stored in the database (encrypted at rest) and can be delivered to external services via the **Worker** webhook.

### Prerequisites

- `acme.enabled` must be set to `true`.
- `acme.email` must be a valid contact address accepted by Let's Encrypt.
- Start with `acme.staging: true` to use the Let's Encrypt staging environment and avoid rate limits while testing.  Switch to `false` for trusted production certificates.
- The Hetzner API token associated with a record must have **write** access to the DNS zone so the challenge TXT record can be created and removed.

### ACME account storage

HDNS persists the ACME account key and registration under the application data directory:

```
<data-dir>/acme/staging/account.key   # EC P-256 private key (PEM, mode 0600)
<data-dir>/acme/staging/account.json  # Let's Encrypt registration resource
<data-dir>/acme/production/account.key
<data-dir>/acme/production/account.json
```

## Worker

The **HDNS Worker** is a lightweight webhook receiver that acts on certificate events emitted by the main server.  It lets you automatically deploy freshly issued certificates to other services running on the same host or on a remote machine.

### Configuration

```yaml
log_level: 1
port: 8080
secret: "<shared-bearer-token>"
tasks:
  - name: "deploy-nginx"
    actions:
      - type: cert_save
        cert_dir: /etc/nginx/certs
        cert_file: fullchain.pem
        key_file: privkey.pem
        # Optional for PKCS#12 consumers:
        # pkcs12_file: certificate.p12
      - type: service_restart
        service_name: nginx
```

### Webhook endpoint

```
POST /<task-name>
Authorization: Bearer <secret>
Content-Type: application/json

{
  "cert": "<PEM leaf>",
  "chain": "<PEM chain>",
  "fullchain": "<PEM leaf+chain>",
  "private_key": "<PEM private key>",
  "certificate": "<PEM leaf+chain>",
  "certificate_format": "pem|pkcs12",
  "pkcs12": "<base64 PKCS#12, optional>",
  "pkcs12_base64": "<base64 PKCS#12, optional>"
}
```

Every request must carry the configured Bearer token in the `Authorization` header.  HDNS sends this token automatically when delivering a certificate.

### Action types

| Type | Description |
|---|---|
| `cert_save` | Writes configured certificate artifacts from the payload to `cert_dir` (`cert_file`, `chain_file`, `fullchain_file`, `key_file`, optional `pkcs12_file`). |
| `service_restart` | Restarts the named OS service (`systemctl restart` on Linux, `sc stop/start` on Windows). |
| `exec` | Runs an arbitrary shell command (`sh -c` on Linux, `cmd /C` on Windows). |

Actions in a task are executed sequentially; the first failure aborts the chain and returns HTTP 500.
