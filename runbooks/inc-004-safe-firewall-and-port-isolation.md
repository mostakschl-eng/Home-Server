# 📋 [INC-004] Safe Firewall Hardening & Dual-Network Access Strategy

---

## 📌 Metadata
- **Incident / SOP ID**: `INC-004`
- **Component**: UFW (Uncomplicated Firewall), Docker IPTables, Tailscale
- **Target Policy**: Allow SSH over both Local Wi-Fi LAN (`192.168.0.x`) and Tailscale (`100.81.129.68`), while isolating management ports
- **Classification**: Standard Operating Procedure (SOP) & Security Hardening
- **Safety Level**: ⚠️ High Prudence (Must execute with auto-rollback failsafe when remote)

---

## 1. 🎯 Objectives & Design Principles

1. **Dual-Path Redundant SSH Access**:
   - SSH (`port 22`) must be reachable from **both** the home Wi-Fi network (`192.168.0.0/24`) and the remote Tailscale WireGuard mesh (`100.64.0.0/10`).
   - If Tailscale ever crashes, local Wi-Fi SSH provides an immediate fail-safe.
2. **Zero Lockout Guarantee**:
   - Any firewall activation must utilize a **timed failsafe rollback** (`sleep 60 && ufw disable`) to prevent being permanently locked out if a rule blocks remote traffic.
3. **Docker NAT Integrity**:
   - By default, Docker writes directly to `iptables` and bypasses naive UFW rules. Firewall rules must explicitly handle Docker packet forwarding without breaking Traefik or Coolify.

---

## 2. 🛡️ Safe Step-by-Step Configuration

### Step 1: Pre-Configure UFW Allow Rules (Before Enabling)
Execute the following commands to ensure all management paths are permitted:

```bash
# Reset default policies safely
sudo ufw default deny incoming
sudo ufw default allow outgoing

# Rule 1: Allow SSH from ANY interface (Both Local Wi-Fi and Tailscale)
sudo ufw allow 22/tcp comment "SSH access on LAN and Tailscale"

# Rule 2: Allow all traffic on Tailscale mesh interface
sudo ufw allow in on tailscale0 comment "Full access over encrypted Tailscale mesh"

# Rule 3: Allow Tailscale WireGuard UDP transport port
sudo ufw allow 41641/udp comment "Tailscale WireGuard direct communication"

# Rule 4: Allow Traefik Web Ingress (HTTP & HTTPS)
sudo ufw allow 80/tcp comment "Traefik HTTP"
sudo ufw allow 443/tcp comment "Traefik HTTPS"
sudo ufw allow 443/udp comment "Traefik HTTP/3 QUIC"

# Rule 5: Allow local loopback
sudo ufw allow in on lo comment "Local loopback"
```

### Step 2: The Fail-Safe Arming Command (Crucial when Remote)
When you enable UFW while away from home, run it with an automatic 60-second emergency rollback:

```bash
sudo sh -c 'ufw --force enable && sleep 60 && ufw disable'
```

- **If your session remains active**: Press `Ctrl+C` to cancel the rollback and keep UFW enabled (`sudo ufw --force enable`).
- **If your connection is interrupted**: The timer expires in 60 seconds and disables UFW automatically, instantly restoring access!

---

## 3. 🔍 Verification & Testing

```bash
# Verify active firewall status
sudo ufw status numbered

# Test SSH connection from Tailscale
ssh mostak@100.81.129.68

# Test SSH connection from Local LAN (when home)
ssh mostak@192.168.0.149
```

---

## 4. 🐳 Docker Compatibility Note
If you notice Docker container port forwarding issues after enabling UFW, ensure `DEFAULT_FORWARD_POLICY="ACCEPT"` in `/etc/default/ufw` or utilize the official Docker UFW helper:
```bash
sudo sed -i 's/DEFAULT_FORWARD_POLICY="DROP"/DEFAULT_FORWARD_POLICY="ACCEPT"/' /etc/default/ufw
sudo ufw reload
```
