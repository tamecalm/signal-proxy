# Production Deployment Guide: zignal.site

Follow these steps to migrate from development to production using your new domain `zignal.site`.

---

## 1. DNS Records (Namecheap)
Add these records in **Advanced DNS**:

| Type | Host | Value | TTL |
|------|------|-------|-----|
| A Record | `@` | *[Vercel IP from Step 2]* | Automatic |
| CNAME | `www` | `cname.vercel-dns.com` | Automatic |
| A Record | `proxy` | `63.178.89.189` | 5 min |
| A Record | `api` | `63.178.89.189` | 5 min |

---

## 2. Frontend Configuration (Vercel)
1. In Vercel, go to **Settings > Domains**.
2. Add `zignal.site`.
3. Copy the IP address Vercel provides (usually `76.76.21.21`).
4. Update the Namecheap `@` A Record with that IP.

---

## 3. EC2 Configuration (Terminal)

### Step A: Stop Services & Get SSL
```bash
# Connect to EC2
ssh -i ~/.ssh/reactra.pem ubuntu@63.178.89.189

# Stop services to free up port 80/443
sudo systemctl stop signal-proxy nginx

# Get certificates for BOTH proxy and api subdomains
sudo certbot certonly --standalone \
  -d proxy.zignal.site \
  -d api.zignal.site \
  --agree-tos \
  --email your@email.com \
  --non-interactive
```

### Step B: Configure SNI Multiplexing
Since both the Signal Proxy and Nginx want port 443, we use Nginx as a "traffic director".

**1. Install the Nginx Stream Module:**
On Ubuntu, the stream module is often separate. Run this first:
```bash
sudo apt update
sudo apt install libnginx-mod-stream -y
```

**2. Create the Stream Config:**
`sudo nano /etc/nginx/nginx.conf`

> [!IMPORTANT]
> The `stream` block **MUST** be at the very end of the file, **OUTSIDE** and **AFTER** the `http { ... }` block. If you put it inside, Nginx will fail.

Scroll to the bottom of the file (after the last `}`) and paste:

```nginx
stream {
    map $ssl_preread_server_name $backend_name {
        api.zignal.site    nginx_api;
        default            signal_proxy;
    }

    upstream nginx_api {
        server 127.0.0.1:4443;
    }

    upstream signal_proxy {
        server 127.0.0.1:10443;
    }

    server {
        listen 443;
        proxy_pass $backend_name;
        ssl_preread on;
    }
}
```

### Step C: Update Signal Proxy Config
Edit `~/signal-proxy/config.json`:
- Change `"listen": ":443"` to `"listen": "127.0.0.1:10443"`
- Ensure `cert_file` is `/etc/letsencrypt/live/proxy.zignal.site/fullchain.pem`
- Ensure `key_file` is `/etc/letsencrypt/live/proxy.zignal.site/privkey.pem`

Update `~/signal-proxy/.env`:
```bash
APP_ENV=production
DOMAIN=proxy.zignal.site
API_DOMAIN=api.zignal.site
ALLOWED_ORIGIN=https://zignal.site
BASE_URL=https://proxy.zignal.site
```

### Step D: Update Nginx API Config
Create/Update `/etc/nginx/sites-available/api.zignal.site`:
```nginx
server {
    listen 127.0.0.1:4443 ssl http2;
    server_name api.zignal.site;

    ssl_certificate /etc/letsencrypt/live/proxy.zignal.site/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/proxy.zignal.site/privkey.pem;

    location /api/ {
        proxy_pass http://127.0.0.1:9090/api/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    location /health {
        return 200 'OK';
        add_header Content-Type text/plain;
    }
}
```

### Step E: Enable and Restart
```bash
# Enable Nginx site
sudo ln -sf /etc/nginx/sites-available/api.zignal.site /etc/nginx/sites-enabled/

# Reload and Restart
sudo systemctl daemon-reload
sudo systemctl restart signal-proxy
sudo nginx -t && sudo systemctl restart nginx
```

---

## 4. Verification
```bash
# 1. Check if Signal Proxy is running
sudo systemctl status signal-proxy

# 2. Check port 443 is handled by Nginx
sudo ss -tlnp | grep 443

# 3. Test API endpoint
curl https://api.zignal.site/api/stats
```
