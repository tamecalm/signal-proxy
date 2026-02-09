# Zignal Proxy Documentation

## Quick Links

| Topic | Document |
|-------|----------|
| **Getting Started** | [HTTPS_PROXY.md](HTTPS_PROXY.md) |
| **Architecture** | [ARCHITECTURE.md](ARCHITECTURE.md) |
| **AWS Deployment** | [aws/DEPLOYMENT.md](aws/DEPLOYMENT.md) |

## Endpoints

| Endpoint | URL |
|----------|-----|
| Signal Proxy | `proxy.zignal.site:8443` |
| HTTP Proxy | `private.zignal.site:8080` |
| SOCKS5 Proxy | `private.zignal.site:1080` |
| PAC File | `https://private.zignal.site/proxy.pac` |

## Documentation Index

### Core Guides
- [HTTPS_PROXY.md](HTTPS_PROXY.md) - HTTP/HTTPS/SOCKS5 proxy quick start
- [ARCHITECTURE.md](ARCHITECTURE.md) - System design and components

### API Reference
- [api/PAC.md](api/PAC.md) - PAC (Proxy Auto-Config) endpoint
- [api/METRICS.md](api/METRICS.md) - Prometheus metrics and stats API

### Configuration
- [configuration/CONFIG.md](configuration/CONFIG.md) - Environment variables reference
- [configuration/USERS.md](configuration/USERS.md) - User management and passwords

### Deployment
- [aws/DEPLOYMENT.md](aws/DEPLOYMENT.md) - AWS EC2 deployment
- [gcp/DEPLOYMENT.md](gcp/DEPLOYMENT.md) - Google Cloud deployment
- [nginx/api-config.md](nginx/api-config.md) - Nginx reverse proxy

## Directory Structure

```
docs/
├── README.md              # This file
├── ARCHITECTURE.md        # System architecture
├── HTTPS_PROXY.md         # Quick start guide
├── api/
│   ├── PAC.md            # PAC endpoint
│   └── METRICS.md        # Metrics API
├── configuration/
│   ├── CONFIG.md         # Config reference
│   └── USERS.md          # User management
├── aws/
│   └── DEPLOYMENT.md     # AWS EC2 guide
├── gcp/
│   └── DEPLOYMENT.md     # GCP guide
└── nginx/
    └── api-config.md     # Nginx config
```
