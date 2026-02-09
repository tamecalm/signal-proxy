# Zignal Proxy Architecture

## Overview

Zignal Proxy is a multi-mode proxy server deployed on **AWS EC2** supporting:
- **Signal Proxy Mode** (`proxy.zignal.site`) - Public TLS proxy for Signal messaging
- **Private Proxy Mode** (`private.zignal.site`) - Authenticated HTTPS/SOCKS5 proxy

## Deployment Architecture

```
                    ┌─────────────────────────────────────┐
                    │           AWS EC2 Instance          │
                    │         Ubuntu 24.04 LTS            │
                    │         Elastic IP assigned         │
                    ├─────────────────────────────────────┤
    Internet ──────►│ ┌─────────────────────────────────┐ │
                    │ │        Zignal Proxy             │ │
                    │ │    /opt/proxy/signal-proxy      │ │
                    │ ├─────────────────────────────────┤ │
                    │ │                                 │ │
 proxy.zignal.site  │ │  Signal Mode (:8443)            │──► Signal Servers
      :8443         │ │  • TLS termination              │    chat.signal.org
                    │ │  • SNI-based routing            │    cdn.signal.org
                    │ │  • No authentication            │    etc.
                    │ │                                 │ │
                    │ ├─────────────────────────────────┤ │
                    │ │                                 │ │
private.zignal.site │ │  HTTPS Mode (:8080/:1080)       │──► Internet
      :8080/:1080   │ │  • HTTP/CONNECT proxy           │
                    │ │  • SOCKS5 proxy                 │
                    │ │  • username:password auth       │
                    │ │  • PAC file at /proxy.pac       │
                    │ │                                 │ │
                    │ └─────────────────────────────────┘ │
                    │                                     │
                    │  /opt/proxy/                        │
                    │  ├── signal-proxy (binary)         │
                    │  ├── users.json (credentials)      │
                    │  ├── .env (configuration)          │
                    │  └── certs/                         │
                    │      ├── cert.pem (Let's Encrypt)  │
                    │      └── key.pem                    │
                    └─────────────────────────────────────┘
```

## Component Overview

### Entry Points

| Port | Domain | Service | Auth |
|------|--------|---------|------|
| 8443 | `proxy.zignal.site` | Signal TLS Proxy | None |
| 8080 | `private.zignal.site` | HTTP Proxy + PAC | Basic Auth |
| 1080 | `private.zignal.site` | SOCKS5 Proxy | User/Pass |
| 9090 | Internal | Metrics & Stats API | None |

### Core Modules

| Module | Path | Purpose |
|--------|------|---------|
| `proxy/` | `internal/proxy/` | Signal TLS proxy with SNI routing |
| `httpproxy/` | `internal/httpproxy/` | HTTP/HTTPS forward proxy |
| `socks5/` | `internal/socks5/` | SOCKS5 proxy (RFC 1928/1929) |
| `pac/` | `internal/pac/` | PAC file generation & serving |
| `auth/` | `internal/auth/` | User authentication & rate limiting |
| `config/` | `internal/config/` | Configuration loading |
| `ui/` | `internal/ui/` | CLI formatting & logging |

## Directory Structure

```
/opt/proxy/                    # EC2 deployment location
├── signal-proxy               # Compiled binary
├── users.json                 # User credentials
├── .env                       # Environment config
└── certs/
    ├── cert.pem              # Let's Encrypt fullchain
    └── key.pem               # Let's Encrypt private key
```

```
Zignal-Backend/                # Source code
├── cmd/proxy/main.go         # Entry point (mode switching)
├── internal/
│   ├── auth/                 # Authentication
│   ├── config/               # Configuration
│   ├── httpproxy/            # HTTP proxy
│   ├── pac/                  # PAC file serving
│   ├── proxy/                # Signal proxy
│   ├── socks5/               # SOCKS5 proxy
│   └── ui/                   # CLI formatting
├── docs/                     # Documentation
├── scripts/                  # Build & utility scripts
└── users.json                # User template
```

## Request Flow

### Signal Proxy Mode
```
Signal App → TLS:8443 → SNI Detection → Route to Signal servers → Relay data
```

### HTTPS Proxy Mode
```
Browser → HTTP:8080 → Proxy-Authorization → Validate user → CONNECT → Target
                  ↓
            /proxy.pac → Return PAC file
```

### SOCKS5 Mode
```
Client → TCP:1080 → Negotiate auth → User/Pass → Connect → Target → Relay
```

## Security

- **TLS 1.2+** for all encrypted connections
- **Let's Encrypt** certificates with auto-renewal
- **bcrypt** password hashing (cost 10+)
- **Rate limiting** per user (token bucket)
- **IP whitelisting** (optional)
- **Systemd hardening** (NoNewPrivileges, ProtectSystem)

## Key Features

| Feature | Description |
|---------|-------------|
| Dual-mode | Signal proxy or general HTTPS/SOCKS5 |
| PAC Support | Dynamic `/proxy.pac` with credential embedding |
| Metrics | Prometheus metrics at `:9090/metrics` |
| Stats API | JSON stats at `:9090/api/stats` |
| Auto-renew | Let's Encrypt certificate auto-renewal |
| Graceful shutdown | Clean connection draining on SIGTERM |
