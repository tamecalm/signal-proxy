# Local Testing Guide

How to build, run, and test the Signal Proxy locally.

---

## Prerequisites

- **Go 1.21+** installed
- **OpenSSL** (for generating test certificates)

---

## Quick Start

### 1. Generate Self-Signed Certificates

```bash
openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes -subj "/CN=localhost"
```

### 2. Build the Proxy

```bash
go build ./cmd/proxy
```

### 3. Run the Proxy

```bash
./proxy
```

You should see output like:

```
╭────────────────────────────────────────────────────────────╮
│   ◆ SIGNAL  v1.0.0                                         │
│  Trusted Proxy Service                                      │
╰────────────────────────────────────────────────────────────╯

19:00:00  ✔  Proxy active on :8443
19:00:00  ℹ  Metrics: http://localhost:9090/metrics
```

---

## Configuration

Edit `config.json` to customize:

| Field           | Default    | Description                    |
|-----------------|------------|--------------------------------|
| `listen`        | `:8443`    | Proxy listen address           |
| `cert_file`     | `server.crt` | TLS certificate path         |
| `key_file`      | `server.key` | TLS private key path         |
| `timeout_sec`   | `300`      | Connection timeout (seconds)   |
| `max_conns`     | `1000`     | Max concurrent connections     |
| `metrics_listen`| `:9090`    | Prometheus metrics endpoint    |
| `hosts`         | -          | SNI → backend mapping          |

---

## Testing Endpoints

### Health Check (Metrics)

```bash
curl http://localhost:9090/metrics
```

### TLS Connection Test

```bash
openssl s_client -connect localhost:8443 -servername chat.signal.org
```

### Using curl with SNI

```bash
curl -v --resolve chat.signal.org:8443:127.0.0.1 \
     --cacert server.crt \
     https://chat.signal.org:8443/
```

---

## Metrics

Available at `http://localhost:9090/metrics`:

| Metric                              | Description                    |
|-------------------------------------|--------------------------------|
| `signalproxy_relay_total`           | Total relayed connections      |
| `signalproxy_active_conns`          | Current active connections     |
| `signalproxy_bytes_total`           | Bytes transferred (up/down)    |
| `signalproxy_errors_total`          | Errors by type                 |
| `signalproxy_connection_duration_seconds` | Connection duration histogram |
| `signalproxy_connections_rejected_total` | Rejected connections (capacity) |

---

## Troubleshooting

| Issue                        | Solution                                    |
|------------------------------|---------------------------------------------|
| `certificate file not found` | Run the OpenSSL command to generate certs   |
| `address already in use`     | Change `listen` port in `config.json`       |
| `connection refused`         | Ensure the proxy is running                 |
| `TLS handshake failure`      | Check certificate/key match and validity    |

---

## Graceful Shutdown

Press `Ctrl+C` to initiate graceful shutdown. Active connections drain within 30 seconds.
