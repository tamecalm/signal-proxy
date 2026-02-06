# AWS EC2 Setup Guide (Fresh Instance)

Follow these steps to create a new, dedicated virtual server for your Signal Proxy.

---

## Step 1: Launch Instance
1.  Log in to your [AWS Management Console](https://console.aws.amazon.com/ec2/).
2.  In the search bar at the top, type **EC2** and click on it.
3.  Click the orange **Launch instance** button.

## Step 2: Name and OS
1.  **Name:** `signal-proxy-server`
2.  **Application and OS Images (Amazon Machine Image):** 
    *   Select **Ubuntu**.
    *   Ensure "Ubuntu Server 22.04 LTS" (Free tier eligible) is selected.
    *   **Architecture:** 64-bit (x86).

## Step 3: Instance Type
1.  **Instance type:** Select `t3.micro` (or `t2.micro` if t3 is not available in your region). Both are Free Tier eligible.

## Step 4: Key Pair
1.  **Key pair name:** Select your existing key `reactra`. 
    *   *Note: Since you already have `reactra.pem` on your computer, you don't need to create a new one.*

## Step 5: Network Settings
1.  Click **Edit** on the top right of the Network settings box.
2.  **Auto-assign public IP:** Ensure this is set to **Enable**.
3.  **Firewall (security groups):** Select "Create security group".
4.  **Security group name:** `signal-proxy-sg`
5.  **Inbound Security Group Rules:**
    *   **Rule 1 (SSH):** Port 22, Source: "My IP" (for security) or "Anywhere" (if you travel).
    *   **Rule 2 (HTTPS):** Port 443, Source: "Anywhere (0.0.0.0/0)".
    *   **Rule 3 (HTTP):** Port 80, Source: "Anywhere (0.0.0.0/0)". *This is needed for the Let's Encrypt certificate setup.*

## Step 6: Storage
1.  The default **8 GiB or 20 GiB gp3** is plenty for this project.

## Step 7: Launch
1.  Click **Launch instance** on the right side summary panel.

---

## Step 8: Get your Public IP
1.  Once launched, click on the **Instance ID** (e.g., `i-0abcd1234efgh`).
2.  Look for **Public IPv4 address**. Copy this.
3.  Go to your **No-IP Dashboard** and update your hostname to point to this NEW IP address.

---

## Step 9: Connect via SSH
Open your terminal (PowerShell or CMD on Windows) and run:

```bash
# Remember to go to the folder where your .pem file is stored
ssh -i "C:\Users\UZOR-GWX\.ssh\reactra.pem" ubuntu@<YOUR_NEW_PUBLIC_IP>
```

---

## Next Steps
Once you are logged into the new instance, follow the **[EC2 Deployment Guide](./EC2_DEPLOYMENT.md)** to install the proxy.
