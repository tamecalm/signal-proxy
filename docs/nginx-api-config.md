# Nginx Configuration for Zignal Proxy Stats API

## Deployment Guide

### 1. Bootstrap for SSL (The "Port 80" Method)

If you haven't gotten SSL certificates yet, Nginx will fail to start if the config references missing `.pem` files. Start with a minimal HTTP block:

```nginx
server {
    listen 80;
    server_name api.zignal.site trysignal.zapto.org;
    
    location / {
        return 200 'ready for certbot';
    }
}
```

Enable this, test config, and reload:
```bash
sudo ln -s /etc/nginx/sites-available/api.zignal.site /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

### 2. Get Certificates
Now run Certbot. It will automatically detect your hostnames and configure the SSL for you:
```bash
sudo certbot --nginx -d api.zignal.site -d trysignal.zapto.org
```

### 3. Final Production Configuration
Certbot will update your files, but you should ensure the `/api/` proxy block is present. Your final config in `/etc/nginx/sites-available/api.zignal.site` should look like this:

```nginx
server {
    listen 443 ssl http2;
    server_name api.zignal.site;

    # SSL certificates (Automatically managed by Certbot)
    ssl_certificate /etc/letsencrypt/live/api.zignal.site/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.zignal.site/privkey.pem;
    
    # CORS headers for the React landing page
    add_header Access-Control-Allow-Origin "*" always;
    add_header Access-Control-Allow-Methods "GET, OPTIONS" always;
    add_header Access-Control-Allow-Headers "Content-Type" always;

    # Proxy to backend metrics server (port 9090)
    location /api/ {
        proxy_pass http://127.0.0.1:9090/api/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Verify It Works

```bash
# Test the stats endpoint
curl https://api.zignal.site/api/stats

# Expected response:
# {"totalUsers":0,"activeConnections":0,"uptimeSeconds":123,"dataThroughput":"0 B/s","latency":18,"successRate":100}
```

