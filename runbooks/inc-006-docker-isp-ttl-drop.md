# 📋 [INC-006] Docker Outbound WAN Packet Drop via ISP Anti-Tethering TTL Filter

---

## 📌 Metadata
- **Incident ID**: `INC-006`
- **Date / Timestamp**: `2026-09-23 06:50 UTC`
- **Impacted Systems**: All Docker containers on host (`nextcloud`, `coolify`, `mcp-memory-service`)
- **Symptoms**:
  - Nextcloud initial setup modal: *"Could not fetch list of apps from the App Store"*
  - Coolify unable to curl public APIs
  - MCP Memory container HuggingFace/S3 connection timeouts during startup
- **Severity Level**: High (Container outbound Internet traffic silently dropped)
- **Lead Engineer**: `mostak`
- **Resolution Status**: **Resolved**

---

## 1. ⚠️ Problem Statement & Symptoms
1. **Nextcloud**: After successful initial deployment, Nextcloud displayed an alert: `Could not fetch list of apps from the App Store` upon loading the welcome screen.
2. **MCP Memory Container**: Previously hung or failed when downloading ONNX models from HuggingFace/AWS S3.
3. **Host vs Container Disparity**:
   - Host `curl -I https://apps.nextcloud.com` returned `HTTP/2 200` instantly.
   - Container `curl -I https://apps.nextcloud.com` timed out after 5-8 seconds.
   - Container `ping 8.8.8.8` (ICMP) and `nslookup` (DNS/UDP) worked perfectly.
   - Container `curl http://192.168.0.1` (Local router) worked with `HTTP/1.1 200 OK`.
   - Only container outbound TCP connections to public WAN IPs timed out.

---

## 2. 🔍 Deep Root Cause Analysis (RCA)
- **Packet Inspection with `tcpdump`**:
  - Outgoing SYN packets sent from the host directly had `TTL=64` (default Linux socket TTL).
  - Packets originated inside Docker containers (`10.0.2.4`) were forwarded by the host kernel across the bridge to `wlp2s0`.
  - Standard IPv4 forwarding decrements TTL by 1, resulting in `TTL=63` when the MASQUERADED packet left `wlp2s0`.
- **The Upstream Drop Mechanism**:
  - The local ISP / upstream GPON gateway applies an **Anti-Tethering / Anti-Subnet inspection filter**: any outbound IPv4 packet with a TTL different from 64 or 128 is silently discarded by the ISP router.
  - While ICMP (ping) and UDP (DNS) bypassed or were exempt from this strict TCP filter, all outbound TCP SYN packets from containers (`TTL=63`) were silently dropped upstream before receiving SYN-ACK.

---

## 3. 🛠️ Resolution: Kernel TTL Normalization

### Step 1: Normalize Outbound TTL in `mangle` Table
By adding an `iptables` rule to the `POSTROUTING` chain of the `mangle` table for the physical Wi-Fi interface (`wlp2s0`), all forwarded container packets have their TTL reset to `64` prior to leaving the server:

```bash
sudo iptables -t mangle -A POSTROUTING -o wlp2s0 -j TTL --ttl-set 64
```

### Step 2: Verification
Immediately upon adding the rule:
```bash
docker exec nextcloud-syo7laqurhdbmmidrumfckbb curl -I https://apps.nextcloud.com
# HTTP/2 200 OK (returned instantly)
```

### Step 3: Persistence Across Reboots
We updated `/etc/systemd/system/tailscale-gro.service`:

```ini
[Unit]
Description=Network Optimizations (Tailscale UDP GRO, TCP MSS Clamping, ISP TTL Normalization)
After=network.target tailscaled.service

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/sbin/ethtool -K wlp2s0 rx-udp-gro-forwarding on rx-gro-list off
ExecStart=/bin/sh -c 'iptables -t mangle -C FORWARD -p tcp -m tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu 2>/dev/null || iptables -t mangle -A FORWARD -p tcp -m tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu'
ExecStart=/bin/sh -c 'iptables -t mangle -C POSTROUTING -o wlp2s0 -j TTL --ttl-set 64 2>/dev/null || iptables -t mangle -A POSTROUTING -o wlp2s0 -j TTL --ttl-set 64'

[Install]
WantedBy=multi-user.target
```

Reloaded and verified:
```bash
sudo systemctl daemon-reload
sudo systemctl restart tailscale-gro
sudo systemctl is-active tailscale-gro # active
```
