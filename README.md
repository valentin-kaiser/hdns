<p align="center">
  <img src="frontend/src/assets/hdns.png" width="365">
</p>

# HDNS - Hetzner Dynamic DNS Management

A modern web-based Dynamic DNS management solution specifically designed for Hetzner DNS services. 
HDNS provides an intuitive interface for managing DNS records and automatically updating them with your current IP address.

Uses [Hetzner's Cloud API v2](https://pkg.go.dev/github.com/hetznercloud/hcloud-go/v2/hcloud) for seamless integration with Hetzner DNS services.

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
- `HDNS_IPV4_RESOLVERS`: Comma-separated list of IPv4 resolvers to determine public IP address
- `HDNS_IPV6_RESOLVERS`: Comma-separated list of IPv6 resolvers to determine public IP address
- `HDNS_DATABASE`: Database connection DSN 

### File-Based Configuration

```yaml
log_level: 1
web_port: 443
certificate_path: "/path/to/cert.pem"
key_path: "/path/to/key.pem"
refresh_cron: "0 * * * *"
dns_servers:
  - "hydrogen.ns.hetzner.com:53"
  - "oxygen.ns.hetzner.com:53"
  - "helium.ns.hetzner.de:53"
  - "ns3.second-ns.de:53"
  - "ns1.your-server.de:53"
  - "ns.second.ns.com:53"
  - "9.9.9.9:53"
  - "1.1.1.1:53"
  - "8.8.8.8:53"
ipv4_resolvers:
  - "https://api.ipify.org"
  - "https://api.my-ip.io/ip"
  - "https://api.ipy.ch"
  - "https://ident.me/"
  - "https://ifconfig.me/ip"
  - "https://icanhazip.com/"
ipv6_resolvers:
  - "https://api6.ipify.org"
  - "https://ipv6.icanhazip.com/"
database: "hdns:hdns@tcp(localhost:3306)/hdns?parseTime=true"
```
