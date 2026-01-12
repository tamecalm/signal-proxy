# Cloudflare Tunnel Development Setup

Cloudflare Tunnel (`cloudflared`) is a **free** alternative to ngrok for exposing your local development server with a public HTTPS URL.

> **📋 Quick Start**: Get a public URL for Signal testing in 5 minutes.

---

## Prerequisites

- Cloudflare account ([sign up free](https://dash.cloudflare.com/sign-up))
- `cloudflared` CLI installed
- For persistent tunnels: A domain added to Cloudflare

---

## Installation

### Windows

```powershell
# Using winget (recommended)
winget install --id Cloudflare.cloudflared

# Or using Chocolatey
choco install cloudflared

# Or using Scoop
scoop install cloudflared

# Or download directly from:
# https://github.com/cloudflare/cloudflared/releases/latest
# Download cloudflared-windows-amd64.exe and rename to cloudflared.exe
```

### macOS

```bash
# Using Homebrew
brew install cloudflare/cloudflare/cloudflared
```

### Linux

```bash
# Debian/Ubuntu
curl -L --output cloudflared.deb https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
sudo dpkg -i cloudflared.deb

# Or using package manager (if available)
# Check: https://pkg.cloudflare.com/

# Or download binary directly
curl -L --output cloudflared https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64
chmod +x cloudflared
sudo mv cloudflared /usr/local/bin/
```

---

## Quick Tunnel (Easiest - No Config Required)

For quick testing, use a temporary tunnel with a random subdomain:

```bash
# Start your Signal Proxy first
./signal-proxy

# In another terminal, create a quick tunnel
cloudflared tunnel --url https://localhost:8443 --no-tls-verify
```

Cloudflared will output something like:

```
Your quick Tunnel has been created! Visit it at:
https://random-words-here.trycloudflare.com
```

Use this URL in your `.env`:

```bash
TUNNEL_PROVIDER=cloudflare
CLOUDFLARE_ENABLED=true
CLOUDFLARE_DOMAIN=random-words-here.trycloudflare.com
```

> **⚠️ Note**: Quick tunnels use random URLs that change each time you restart. For persistent URLs, use Named Tunnels below.

---

## Named Tunnels (Persistent - Recommended)

Named tunnels provide stable hostnames tied to your Cloudflare domain.

### 1. Authenticate with Cloudflare

```bash
cloudflared tunnel login
```

This opens a browser to authorize `cloudflared` with your Cloudflare account. A certificate is saved to `~/.cloudflared/cert.pem`.

### 2. Create a Tunnel

```bash
cloudflared tunnel create signal-proxy
```

This creates a tunnel and saves credentials to `~/.cloudflared/<TUNNEL_ID>.json`.

Note the Tunnel ID output — you'll need it for configuration.

### 3. Configure DNS

Add a CNAME record pointing your subdomain to the tunnel:

```bash
cloudflared tunnel route dns signal-proxy signal-dev.yourdomain.com
```

Or manually in Cloudflare Dashboard:
- **Type**: CNAME
- **Name**: `signal-dev` (or your preferred subdomain)
- **Target**: `<TUNNEL_ID>.cfargotunnel.com`
- **Proxy status**: Proxied (orange cloud)

### 4. Create Configuration File

```bash
cd cloudflare
cp config.yml.example config.yml
# Edit config.yml with your settings
```

Update `config.yml`:

```yaml
tunnel: <YOUR_TUNNEL_ID>
credentials-file: C:\Users\YourUser\.cloudflared\<TUNNEL_ID>.json

ingress:
  - hostname: signal-dev.yourdomain.com
    service: https://localhost:8443
    originRequest:
      noTLSVerify: true
  - service: http_status:404
```

### 5. Start the Tunnel

```bash
# Using config file
cloudflared tunnel --config cloudflare/config.yml run signal-proxy

# Or run by tunnel name (if config is in default location)
cloudflared tunnel run signal-proxy
```

### 6. Update Environment

```bash
TUNNEL_PROVIDER=cloudflare
CLOUDFLARE_ENABLED=true
CLOUDFLARE_DOMAIN=signal-dev.yourdomain.com
```

---

## Development Workflow

### Standard Workflow

```bash
# Terminal 1: Start Cloudflare Tunnel
cloudflared tunnel --url https://localhost:8443 --no-tls-verify

# Terminal 2: Start Signal Proxy
# (update .env with tunnel URL first)
cd signal-proxy
./signal-proxy
```

### With Named Tunnel

```bash
# Terminal 1: Start Named Tunnel
cloudflared tunnel --config cloudflare/config.yml run signal-proxy

# Terminal 2: Start Signal Proxy
./signal-proxy
```

### Environment Setup

Your `.env` should look like:

```bash
APP_ENV=development
DEBUG=true
LOG_LEVEL=debug

# Tunnel Configuration
TUNNEL_PROVIDER=cloudflare

# Cloudflare Tunnel Configuration
CLOUDFLARE_ENABLED=true
CLOUDFLARE_DOMAIN=signal-dev.yourdomain.com

# Domain (uses Cloudflare domain in development)
DOMAIN=signal-dev.yourdomain.com
BASE_URL=https://signal-dev.yourdomain.com
```

---

## Troubleshooting

### "failed to connect to origin" or "bad certificate"

**Cause**: Cloudflared can't verify your local server's self-signed certificate.

**Solution**: Add `--no-tls-verify` flag or set `noTLSVerify: true` in config:

```bash
cloudflared tunnel --url https://localhost:8443 --no-tls-verify
```

Or in `config.yml`:

```yaml
ingress:
  - hostname: your.domain.com
    service: https://localhost:8443
    originRequest:
      noTLSVerify: true
```

### "tunnel credentials file not found"

**Cause**: Missing credentials from `cloudflared tunnel create`.

**Solution**: Run `cloudflared tunnel login` first, then create the tunnel.

### "failed to fetch quick Tunnel"

**Cause**: Network issues or Cloudflare API problems.

**Solution**: 
- Check internet connection
- Try again in a few minutes
- Use a VPN if your network blocks Cloudflare

### Tunnel Stops After Closing Terminal

**Cause**: Tunnel process terminated with terminal.

**Solution**: Run as a background service or use a terminal multiplexer:

```bash
# Windows (PowerShell)
Start-Process cloudflared -ArgumentList "tunnel","--url","https://localhost:8443","--no-tls-verify" -WindowStyle Hidden

# macOS/Linux
nohup cloudflared tunnel --url https://localhost:8443 --no-tls-verify &
```

---

## Quick Tunnel vs Named Tunnel

| Feature | Quick Tunnel | Named Tunnel |
|---------|-------------|--------------|
| Setup | Instant | 5-10 minutes |
| URL | Random, changes each time | Stable, your domain |
| DNS Config | Not needed | Required |
| Cloudflare Account | Optional | Required |
| Best For | Quick testing | Consistent development |

---

## Security Notes

> **⚠️ Warning**: Tunnels expose your local server to the internet.

- Never run with production data
- Use Cloudflare Access for authentication if needed
- Monitor tunnel traffic in Cloudflare Dashboard
- Stop tunnels when not in use

---

## Useful Commands

```bash
# Check cloudflared version
cloudflared --version

# List all tunnels
cloudflared tunnel list

# Get tunnel info
cloudflared tunnel info signal-proxy

# Delete a tunnel
cloudflared tunnel delete signal-proxy

# View tunnel logs
cloudflared tunnel --loglevel debug --url https://localhost:8443
```

---

## Resources

- [Cloudflare Tunnel Documentation](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/)
- [cloudflared GitHub Releases](https://github.com/cloudflare/cloudflared/releases)
- [Cloudflare Dashboard](https://dash.cloudflare.com/)
- [Tunnel Configuration Reference](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/configure-tunnels/local-management/configuration-file/)
