# No-IP Configuration Guide for Signal Proxy

This guide walks you through setting up your No-IP hostname to point to your AWS EC2 instance.

---

## Step 1: Create a New Hostname
1.  Log in to your [No-IP Dashboard](https://www.noip.com/login).
2.  Click on **Dynamic DNS** in the left sidebar.
3.  Click on **Hostnames**.
4.  Click the orange **Create Hostname** button.

## Step 2: Configure Hostname Details
1.  **Hostname:** Enter your preferred name (e.g., `mysignalproxy`).
2.  **Domain:** Select a domain from the dropdown (e.g., `ddns.net`, `serveblog.net`, etc.). 
    *   *Note: Using a simple one like `ddns.net` is common.*
3.  **Record Type:** Ensure **DNS Host (A)** is selected.
4.  **IPv4 Address:** Enter your Elastic IP address: **`63.178.89.189`**.
5.  Click **Create Hostname** at the bottom.

## Step 3: Verify the Configuration
1.  Once created, you will see your new hostname in the list (e.g., `mysignalproxy.ddns.net`).
2.  It may take a few minutes for the DNS to propagate globally.
3.  You can verify it from your local computer's terminal:
    ```bash
    nslookup mysignalproxy.ddns.net
    ```
    *It should return your EC2 IP: `63.178.89.189`.*

---

## Important Tips for Signal Proxy

### 1. Dedicated Domain
If you have multiple projects on the same EC2 instance, it is best to use a **unique subdomain** for the Signal Proxy (e.g., `signal.yourdomain.ddns.net`). This allows you to manage certificates and traffic more easily.

### 2. Monitoring
No-IP free accounts require you to "confirm" your hostname every 30 days via email. Keep an eye on your inbox so your proxy doesn't go offline!

### 3. Usage in Signal
Once verified, your proxy URL for the Signal app will be:
`https://yourhostname.ddns.net`

---

## Next Steps
Now that your domain is pointing to your server, return to your SSH terminal and proceed with **Step 4: Get Let's Encrypt Certificate** in the [EC2 Deployment Guide](./EC2_DEPLOYMENT.md).
