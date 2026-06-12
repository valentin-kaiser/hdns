<p align="center">
  <img src="frontend/src/assets/hdns.png" width="365">
</p>

# HDNS - Hetzner Dynamic DNS Management

HDNS is a self-hosted web application for managing DNS records on Hetzner DNS. It keeps your A/AAAA records in sync with your current public IP address and handles the full lifecycle of Let's Encrypt TLS certificates using the DNS-01 challenge, with no extra open ports required.

Built on [Hetzner's Cloud API v2](https://pkg.go.dev/github.com/hetznercloud/hcloud-go/v2/hcloud), HDNS publishes ACME challenge TXT records directly through the Hetzner DNS API. Certificates are stored encrypted in the database and can be delivered to other services automatically via the accompanying Worker.

## Installation

The quickest way to run HDNS is with Docker Compose:

```bash
git clone https://github.com/valentin-kaiser/hdns.git
cd hdns
docker compose up -d
```

This starts HDNS on port 443 together with a MariaDB instance. Data is persisted in `./data` and `db-data` volumes.

To run just the main server from a pre-built image:

```bash
docker pull ghcr.io/valentin-kaiser/hdns:latest
docker run -p 443:443 -v ./data:/app/data ghcr.io/valentin-kaiser/hdns:latest
```

To build from source:

```bash
git clone https://github.com/valentin-kaiser/hdns.git
cd hdns
docker build --tag hdns .
docker run -p 443:443 -v ./data:/app/data hdns
```

## Configuration

### Environment Variables

| Variable | Description |
|---|---|
| `HDNS_LOG_LEVEL` | Log level: `0`=debug, `1`=info, `2`=warn, `3`=error, `4`=fatal, `5`=panic |
| `HDNS_WEB_PORT` | Port to bind the web server to |
| `HDNS_CERTIFICATE_PATH` | Path to the TLS certificate file |
| `HDNS_KEY_PATH` | Path to the TLS private key file |
| `HDNS_REFRESH_CRON` | Cron expression for the DDNS refresh job |
| `HDNS_DNS_SERVERS` | Comma-separated list of DNS servers used for propagation checks |
| `HDNS_IPV4_RESOLVERS` | Comma-separated resolvers to detect the current public IPv4. Supports `http(s)://` and `dns://host/name?type=A` URIs |
| `HDNS_IPV6_RESOLVERS` | Same format as `HDNS_IPV4_RESOLVERS`; default query type is `AAAA` |
| `HDNS_DATABASE` | MariaDB connection DSN |
| `HDNS_ACME_ENABLED` | Set to `true` to enable Let's Encrypt certificate issuance |
| `HDNS_ACME_EMAIL` | Contact address registered with the ACME CA |
| `HDNS_ACME_STAGING` | Set to `true` to use the Let's Encrypt staging environment (default: `true`) |
| `HDNS_ACME_RENEW_BEFORE_DAYS` | Days before expiry at which renewal is triggered (default: `30`) |
| `HDNS_ACME_RENEW_CRON` | Cron expression for the renewal scan (default: `0 3 * * *`) |

### File-Based Configuration

Place a `hdns.yaml` file in the data directory. All keys map directly to environment variables without the `HDNS_` prefix.

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
  - 9.9.9.9:53
  - 1.1.1.1:53
  - 8.8.8.8:53
ipv4resolvers:
  - dns://resolver1.opendns.com/myip.opendns.com?type=A
  - dns://ns1.google.com/o-o.myaddr.l.google.com?type=TXT
  - dns://1.1.1.1/whoami.cloudflare?type=TXT&class=CH
  - https://icanhazip.com/
ipv6resolvers:
  - dns://resolver1.opendns.com/myip.opendns.com?type=AAAA
  - dns://[2606:4700:4700::1111]/whoami.cloudflare?type=TXT&class=CH
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

HDNS issues and renews TLS certificates from Let's Encrypt using the DNS-01 challenge via the Hetzner DNS API. No extra open ports or publicly reachable HTTP endpoints are needed.

### How it works

1. Configure a DNS record with purpose **Certificate** or **Both** (DDNS + certificate).
2. HDNS creates a temporary `_acme-challenge` TXT record in the Hetzner zone.
3. After the ACME CA verifies the challenge, the signed certificate and key are stored in the database (encrypted at rest) and written to disk.
4. Certificate delivery to other services is handled by the Worker via webhook.

Issuance can be triggered manually from the UI or runs automatically on the renewal schedule.

### Prerequisites

- Set `acme.enabled: true`.
- Set `acme.email` to a valid address accepted by Let's Encrypt.
- Use `acme.staging: true` while testing to stay within Let's Encrypt rate limits. Switch to `false` for production certificates.
- The Hetzner API token for each record needs write access to the DNS zone.

### Per-record ACME account

Each DNS record can specify its own ACME account email under the record settings. When set, HDNS uses that address for all issuance requests for that record. Records without an explicit email fall back to the global `acme.email` value.

Account keys and registrations are stored separately per email address:

```
<data-dir>/acme/staging/<email>/account.key
<data-dir>/acme/staging/<email>/account.json
<data-dir>/acme/production/<email>/account.key
<data-dir>/acme/production/<email>/account.json
```

### Issuance log

Each certificate issuance job records the full lego/ACME client log. The log is available in the certificate details view in the UI and is also written to `<data-dir>/logs/acme.log`. During manual issuance from the UI, log lines are streamed live.

## Worker

The Worker is a lightweight webhook receiver that acts on certificate and IP-change events from the main server. Run it on the target host to deploy certificates automatically after issuance.

### Running with Docker Compose

The Worker is included in the default `compose.yml`. Set the shared secret and map a data directory:

```yaml
hdns-worker:
  build: ./worker
  ports:
    - "8080:8080"
  volumes:
    - ./data/worker:/app/data
  environment:
    - HDNS_WORKER_SECRET=your-secret-here
```

Or run it as a standalone binary or system service on the target host.

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
  "certificate_format": "pem|pkcs12",
  "pkcs12_base64": "<base64 PKCS#12, optional>"
}
```

HDNS sends the Bearer token automatically when delivering a certificate.

### Action types

| Type | Description |
|---|---|
| `cert_save` | Writes certificate artifacts from the payload to `cert_dir`. Supports `cert_file`, `chain_file`, `fullchain_file`, `key_file`, and optional `pkcs12_file`. |
| `service_restart` | Restarts the named OS service via `systemctl` on Linux or `sc stop/start` on Windows. |
| `exec` | Runs an arbitrary shell command (`sh -c` on Linux, `cmd /C` on Windows). |
| `fortios_upload` | Uploads a PKCS#12 certificate to a FortiGate via REST API. |
| `fortios_profile_cert_replace` | Updates certificate references in FortiOS CMDB profile objects. |
| `fortios_admin_server_cert_update` | Updates the FortiOS admin server certificate. |

Actions inside a task run sequentially. The first failure stops the chain and returns HTTP 500.
