# Signal Proxy

<p align="center">
  <strong>A high-performance TLS proxy for Signal messaging infrastructure</strong>
</p>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#configuration">Configuration</a> •
  <a href="#deployment">Deployment</a> •
  <a href="#contributing">Contributing</a>
</p>

---

Signal Proxy is a privacy-focused TLS proxy designed to route Signal traffic through trusted infrastructure. It enables users in restricted regions to access Signal services by proxying encrypted connections to Signal's servers while preserving end-to-end encryption.

## Features

- **TLS Termination & Re-encryption** — Secure TLS 1.2+ connections with automatic certificate handling
- **SNI-based Routing** — Intelligent routing based on Server Name Indication for multiple Signal endpoints
- **Connection Limiting** — Built-in rate limiting with configurable max connections
- **Graceful Shutdown** — Clean connection draining on shutdown signals
- **Prometheus Metrics** — Production-ready observability with `/metrics` endpoint
- **JSON Stats API** — Real-time server statistics for web frontend via `/api/stats`
- **Historical Data** — 24h usage patterns and traffic history via `/api/history`
- **Beautiful CLI** — Semantic logging with colored output, banners, and status indicators
- **Environment Aware** — First-class support for development and production environments

## Quick Start

### Prerequisites

- Go 1.21 or later
- TLS certificate and key for your domain
- Network access to Signal servers

### Installation

```bash
# Clone the repository
git clone https://github.com/tamecalm/signal-proxy.git
cd signal-proxy

# Build the binary
go build -o signal-proxy ./cmd/proxy

# Run the proxy
./signal-proxy
```

### Basic Usage

1. **Generate or obtain TLS certificates** for your domain:
   ```bash
   # Using Let's Encrypt (recommended for production)
   certbot certonly --standalone -d proxy.yourdomain.com
   
   # Or generate self-signed for development
   openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
     -keyout server.key -out server.crt
   ```

2. **Configure the proxy** by editing `config.json`:
   ```json
   {
     "listen": ":8443",
     "cert_file": "server.crt",
     "key_file": "server.key",
     "hosts": {
       "chat.signal.org": "chat.signal.org:443"
     }
   }
   ```

3. **Start the proxy**:
   ```bash
   ./signal-proxy
   ```

## Configuration

### Config File (`config.json`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `listen` | string | `:8443` | Address and port to listen on |
| `cert_file` | string | `server.crt` | Path to TLS certificate |
| `key_file` | string | `server.key` | Path to TLS private key |
| `timeout_sec` | int | `300` | Connection timeout in seconds |
| `max_conns` | int | `1000` | Maximum concurrent connections |
| `metrics_listen` | string | `:9090` | Prometheus and Stats API endpoint |
| `hosts` | object | `{}` | SNI to upstream host mapping |

### Environment Variables

| Variable | Development Default | Production Default | Description |
|----------|--------------------|--------------------|-------------|
| `APP_ENV` | `development` | `production` | Environment mode |
| `NGROK_ENABLED` | `true` | `false` | Use ngrok tunnel (dev only) |
| `NGROK_DOMAIN` | *(empty)* | N/A | Your ngrok domain |
| `DOMAIN` | ngrok domain or `localhost:8443` | `proxy.yourdomain.com` | Base domain |
| `BASE_URL` | `https://${DOMAIN}` | `https://proxy.yourdomain.com` | Full base URL |
| `DEBUG` | `true` | `false` | Enable debug logging |
| `LOG_LEVEL` | `debug` | `info` | Log verbosity |

### Supported Signal Hosts

The proxy supports all Signal infrastructure endpoints:

```json
{
  "hosts": {
    "chat.signal.org": "chat.signal.org:443",
    "cdn.signal.org": "cdn.signal.org:443",
    "cdn2.signal.org": "cdn2.signal.org:443",
    "cdn3.signal.org": "cdn3.signal.org:443",
    "storage.signal.org": "storage.signal.org:443",
    "sfu.voip.signal.org": "sfu.voip.signal.org:443",
    "updates.signal.org": "updates.signal.org:443",
    "directory.signal.org": "directory.signal.org:443",
    "backup.signal.org": "backup.signal.org:443"
  }
}
```

## Deployment

### Development (with ngrok)

> **⚠️ Important**: Signal doesn't allow localhost for testing. You must use ngrok to expose your local server with a public HTTPS URL.

**Quick Start:**

```bash
# 1. Start ngrok tunnel (in Terminal 1)
ngrok tls 8443

# 2. Copy the ngrok URL (e.g., abc123xyz.ngrok.io)

# 3. Create .env file with ngrok domain
cp env.development.example .env
# Edit .env and set NGROK_DOMAIN=abc123xyz.ngrok.io

# 4. Start the proxy (in Terminal 2)
./signal-proxy
```

**With ngrok config file:**

```bash
# Terminal 1
cd ngrok
ngrok start --config ngrok.yml signal-proxy

# Terminal 2  
./signal-proxy
```

See [ngrok/README.md](ngrok/README.md) for detailed setup instructions.

### Production

```bash
# Set production environment
APP_ENV=production \
DOMAIN=proxy.yourdomain.com \
./signal-proxy
```

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o signal-proxy ./cmd/proxy

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/signal-proxy .
COPY config.json .
EXPOSE 8443 9090
CMD ["./signal-proxy"]
```

### Systemd Service

```ini
[Unit]
Description=Signal Proxy
After=network.target

[Service]
Type=simple
User=signal-proxy
WorkingDirectory=/opt/signal-proxy
ExecStart=/opt/signal-proxy/signal-proxy
Restart=always
RestartSec=5
Environment=APP_ENV=production
Environment=DOMAIN=proxy.yourdomain.com

[Install]
WantedBy=multi-user.target
```

## Metrics

The proxy exposes Prometheus metrics at `http://localhost:9090/metrics`:

| Metric | Type | Description |
|--------|------|-------------|
| `signalproxy_active_conns` | Gauge | Current active connections |
| `signalproxy_relay_total` | Counter | Total relayed connections by SNI |
| `signalproxy_bytes_total` | Counter | Total bytes transferred by direction |
| `signalproxy_errors_total` | Counter | Total errors by type |

### JSON API

The proxy also provides a JSON API for frontend integration (Port 9090):

- **Stats**: `GET /api/stats` — Real-time telemetry (uptime, throughput, success rate)
- **History**: `GET /api/history` — 24-hour historical usage data

## Signal Client Configuration

To use this proxy with the Signal app, users need to configure their Signal client to route traffic through your proxy domain. See [Signal's proxy documentation](https://signal.org/blog/run-a-proxy/) for client-side setup instructions.

## Security Considerations

> **Important**: This proxy handles TLS traffic. Ensure your deployment follows security best practices:

- Use valid TLS certificates from a trusted CA
- Keep certificates and private keys secure
- Run the proxy with minimal privileges
- Enable firewall rules to restrict access
- Monitor metrics for anomalies
- Keep the proxy updated

## Contributing

We welcome contributions from the community. To contribute:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please ensure your code:
- Follows the existing code style
- Includes appropriate tests
- Updates documentation as needed

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Signal](https://signal.org) for creating privacy-focused communication
- The open-source community for proxy infrastructure inspiration