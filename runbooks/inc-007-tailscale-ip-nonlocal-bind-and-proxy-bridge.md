# 📋 [INC-007] Tailscale Non-Local Bind Failure on Boot & Traefik Service Network Disconnect

---

## 📌 Metadata
- **Incident ID**: `INC-007`
- **Date / Timestamp**: `2026-09-23 13:55 UTC`
- **Impacted Service(s)**: Coolify PaaS, Nextcloud Suite, Traefik Reverse Proxy (`coolify-proxy`), Coolify Sentinel
- **Severity Level**: High (Both Coolify and Nextcloud web interfaces inaccessible from browser / returning 404)
- **Lead Engineer**: `mostak`
- **Resolution Status**: **Resolved**

---

## 1. ⚠️ Problem Statement & Symptoms

1. **User Observation**:
   - Accessing `http://coolify.100.81.129.68.sslip.io` returned a Traefik plain-text `404 page not found`.
   - Accessing `http://nextcloud.100.81.129.68.sslip.io` returned `404 page not found` or connection timeout.
   - Accessing `http://100.81.129.68:8000` failed to connect.

2. **Container Status at Triage**:
   - `coolify`: Exited with code `255` (`failed to set up container networking: driver failed programming external connectivity on endpoint coolify: failed to bind host port 100.81.129.68:8000/tcp: cannot assign requested address`).
   - `coolify-sentinel`: Exited with code `255`.
   - `coolify-proxy`: Running, but only attached to the default `coolify` Docker network (`10.0.1.0/24`).
   - `nextcloud-syo7laqurhdbmmidrumfckbb`: Running inside its private bridge network `syo7laqurhdbmmidrumfckbb` (`10.0.2.0/24`), completely unreachable from Traefik.

---

## 2. 🔍 Root Cause Analysis (RCA)

### Root Cause 1: Linux Kernel Non-Local IP Binding Blocked on Boot
- When the server rebooted, Docker Engine started before Tailscale had finished initializing and assigning `100.81.129.68` to interface `tailscale0`.
- The Linux kernel parameter `net.ipv4.ip_nonlocal_bind` defaulted to `0` (disabled).
- Docker attempted to start the `coolify` container with port mapping `100.81.129.68:8000:8080`.
- The kernel rejected the socket bind with `EADDRNOTAVAIL: cannot assign requested address`.
- As a consequence, `coolify` crashed with exit code 255 and was left in an unstarted, networkless state.

### Root Cause 2: Reverse Proxy Network Isolation for Nextcloud
- In Coolify's architecture, services (like Nextcloud) are deployed into isolated Docker networks (`syo7laqurhdbmmidrumfckbb`).
- Traefik (`coolify-proxy`) requires dynamic attachment to each service network to route incoming HTTP requests to internal container IPs (`10.0.2.3`).
- Because Coolify was down from the boot failure, Coolify's service orchestrator never attached `coolify-proxy` to `syo7laqurhdbmmidrumfckbb`.
- Traefik dropped the routing rule for `nextcloud.100.81.129.68.sslip.io`, returning `404 page not found`.

---

## 3. 🛠️ Step-by-Step Resolution Procedures

### Step 1: Enable & Persist Non-Local IP Binding in Kernel
Allow sockets to bind to IP addresses that may not yet be active on network interfaces at boot time:

```bash
# Enable in live kernel
sudo sysctl -w net.ipv4.ip_nonlocal_bind=1

# Persist across all reboots
sudo bash -c "echo 'net.ipv4.ip_nonlocal_bind = 1' > /etc/sysctl.d/99-tailscale-bind.conf"
sudo sysctl -p /etc/sysctl.d/99-tailscale-bind.conf
```

### Step 2: Recreate and Start Coolify Stack
Recreate the Coolify container using its official compose specifications:

```bash
cd /data/coolify/source
sudo docker compose --env-file .env -f docker-compose.yml -f docker-compose.prod.yml up -d --force-recreate coolify
sudo docker start coolify-sentinel
```

### Step 3: Attach Traefik to the Nextcloud Service Network
Bridge Traefik into Nextcloud's isolated network so Traefik can route traffic to internal container port 80:

```bash
sudo docker network connect syo7laqurhdbmmidrumfckbb coolify-proxy
```

---

## 4. ✅ Verification & Quality Assurance

Tested connectivity from remote client workstation:

```bash
# 1. Coolify Direct Tailscale Port
curl -I http://100.81.129.68:8000
# Result: HTTP/1.1 302 Found -> /login

# 2. Coolify via Traefik Reverse Proxy
curl -I http://coolify.100.81.129.68.sslip.io
# Result: HTTP/1.1 302 Found -> /login

# 3. Nextcloud via Traefik Reverse Proxy
curl -I http://nextcloud.100.81.129.68.sslip.io
# Result: HTTP/1.1 302 Found -> /login
```

All Docker containers verified `healthy`:
```text
coolify                                 Up (healthy)   100.81.129.68:8000->8080/tcp
coolify-proxy                           Up (healthy)   0.0.0.0:80, 443, 8080
coolify-sentinel                        Up (healthy)
nextcloud-syo7laqurhdbmmidrumfckbb      Up (healthy)   80/tcp, 443/tcp
nextcloud-db-syo7laqurhdbmmidrumfckbb   Up (healthy)   5432/tcp
redis-syo7laqurhdbmmidrumfckbb          Up (healthy)   6379/tcp
coolify-db                              Up (healthy)   5432/tcp
coolify-redis                           Up (healthy)   6379/tcp
coolify-realtime                        Up (healthy)   6001-6002
```

---

## 5. 🛡️ Preventative Measures & Action Items

- [x] Kernel parameter `net.ipv4.ip_nonlocal_bind = 1` persisted in `/etc/sysctl.d/99-tailscale-bind.conf`.
- [x] Traefik network attachment verified and saved in Docker container configuration.
- [x] Documented in `runbooks/inc-007-tailscale-ip-nonlocal-bind-and-proxy-bridge.md`.
- [x] Added changelog entry in `CHANGELOG.md`.
