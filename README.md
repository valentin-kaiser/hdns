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
- `HDNS_IPV4_RESOLVERS`: Comma-separated list of IPv4 resolvers to determine public IP address. Each entry is a URI; `http(s)://…` entries perform an HTTP GET and expect the plain IP in the body, while `dns://<server>[:port]/<query-name>?type=A|AAAA|TXT[&class=IN|CH]` entries perform a DNS query (e.g. `dns://resolver1.opendns.com/myip.opendns.com?type=A`, `dns://1.1.1.1/whoami.cloudflare?type=TXT&class=CH`).
- `HDNS_IPV6_RESOLVERS`: Comma-separated list of IPv6 resolvers to determine public IP address. Same URI format as `HDNS_IPV4_RESOLVERS`; for DNS entries the default query type is `AAAA`.
- `HDNS_DATABASE`: Database connection DSN 

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
- ns.second.ns.com:53
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
```
