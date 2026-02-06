# Signal Proxy EC2 Deployment Guide

> You're SSH'd into EC2. Follow these steps in order.

---

## Step 1: Install Dependencies

```bash
# Update system
sudo yum update -y          # Amazon Linux
# OR
sudo apt update && sudo apt upgrade -y  # Ubuntu

# Install Go 1.21+
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version  # Verify: go1.21.6

# Install certbot for Let's Encrypt
sudo yum install -y certbot   # Amazon Linux
# OR
sudo apt install -y certbot   # Ubuntu
```

---

## Step 2: Configure No-IP DDNS

1. Go to [no-ip.com](https://www.noip.com/) and create a free hostname
2. Point it to your EC2 **public IP**
3. Wait ~5 minutes for DNS propagation
4. Verify: `nslookup yoursubdomain.ddns.net`

---

## Step 3: Open EC2 Security Group Ports

In AWS Console → EC2 → Security Groups → Edit inbound rules:

| Type | Port | Source | Purpose |
|------|------|--------|---------|
| HTTPS | 443 | 0.0.0.0/0 | Signal proxy traffic |
| HTTP | 80 | 0.0.0.0/0 | Let's Encrypt verification (temporary) |
| Custom TCP | 9090 | Your IP | Prometheus metrics (optional) |

---

## Step 4: Get Let's Encrypt Certificate

```bash
# Stop any service on port 80
sudo systemctl stop nginx httpd 2>/dev/null || true

# Get certificate (replace with your domain)
sudo certbot certonly --standalone \
  -d yoursubdomain.ddns.net \
  --agree-tos \
  --email your@email.com \
  --non-interactive

# Verify certificates exist
ls -la /etc/letsencrypt/live/yoursubdomain.ddns.net/
# Should see: fullchain.pem, privkey.pem
```

---

## Step 5: Upload & Build Signal Proxy

### Option A: From Local Machine
```bash
# On your LOCAL machine (not EC2)
cd signal-proxy
go build -o signal-proxy ./cmd/proxy

# Upload to EC2
scp signal-proxy config.production.example.json env.production.example \
  ec2-user@YOUR_EC2_IP:~/signal-proxy/
```

### Option B: Build on EC2
```bash
# On EC2: Clone and build
git clone https://github.com/tamecalm/signal-proxy.git
cd signal-proxy
go build -o signal-proxy ./cmd/proxy
```

---

## Step 6: Configure for Production

```bash
cd ~/signal-proxy

# Create production config
cp config.production.example.json config.json

# Edit config - replace YOUR_DOMAIN with your actual subdomain
nano config.json
```

**config.json** should look like:
```json
{
  "listen": ":443",
  "cert_file": "/etc/letsencrypt/live/yoursubdomain.ddns.net/fullchain.pem",
  "key_file": "/etc/letsencrypt/live/yoursubdomain.ddns.net/privkey.pem",
  "timeout_sec": 300,
  "max_conns": 10000,
  "metrics_listen": "127.0.0.1:9090",
  "hosts": { ... }
}
```

```bash
# Create .env file
cp env.production.example .env
nano .env
```

**Set your domain:**
```bash
APP_ENV=production
DOMAIN=yoursubdomain.ddns.net
BASE_URL=https://yoursubdomain.ddns.net
DEBUG=false
LOG_LEVEL=info
```

---

## Step 7: Test Run

```bash
# Run manually (needs sudo for port 443)
sudo ./signal-proxy

# You should see:
# ✓  Proxy active on :443
# ✓  Certificates reloaded from disk
```

Test from another terminal:
```bash
curl -I https://yoursubdomain.ddns.net --resolve yoursubdomain.ddns.net:443:127.0.0.1 -k
```

Press `Ctrl+C` to stop after testing.

---

## Step 8: Create Systemd Service

```bash
sudo nano /etc/systemd/system/signal-proxy.service
```

Paste:
```ini
[Unit]
Description=Signal Proxy
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/home/ec2-user/signal-proxy
ExecStart=/home/ec2-user/signal-proxy/signal-proxy
Restart=always
RestartSec=5
Environment=APP_ENV=production
Environment=DOMAIN=yoursubdomain.ddns.net

[Install]
WantedBy=multi-user.target
```

> **Note:** Change `/home/ec2-user/` to `/home/ubuntu/` if using Ubuntu.

```bash
# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable signal-proxy
sudo systemctl start signal-proxy

# Check status
sudo systemctl status signal-proxy
```

---

## Step 9: Auto-Renew Certificates

```bash
sudo nano /etc/cron.d/certbot-signal
```

Paste:
```bash
0 0 1 * * root certbot renew --quiet --deploy-hook "systemctl reload signal-proxy || pkill -HUP signal-proxy"
```

---

## Step 10: Verify Deployment

```bash
# Check service is running
sudo systemctl status signal-proxy

# Check port 443 is listening
sudo ss -tlnp | grep 443

# Test TLS connection
openssl s_client -connect yoursubdomain.ddns.net:443 -servername chat.signal.org
```

---

## Share Your Proxy

Give users this link format:
```
https://signal.tube/#yoursubdomain.ddns.net
```

Or in Signal app: **Settings → Data & Storage → Proxy → Add proxy**

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Port 443 in use | `sudo lsof -i :443` then stop conflicting service |
| Certificate errors | Check paths in config.json match `/etc/letsencrypt/live/...` |
| Connection refused | Check Security Group allows port 443 |
| Service won't start | `journalctl -u signal-proxy -f` for logs |
