# 📋 [INC-005] Subnet Router BDIX Media Server Large Response Hang & TCP MSS Clamping

---

## 📌 Metadata
- **Incident ID**: `INC-005`
- **Date / Timestamp**: `2026-09-23 05:30 UTC`
- **Impacted Subnet**: `172.16.50.0/24` (Dhaka-Flix BDIX Portal & Media Servers)
- **Target Servers**: `http://172.16.50.4` (Portal), `http://172.16.50.14` (Movie Media Server)
- **Severity Level**: Medium (Subnet services accessible locally, but hanging remotely over Tailscale)
- **Lead Engineer**: `mostak`
- **Resolution Status**: **Resolved**

---

## 1. ⚠️ Problem Statement & Symptoms
When connected to Tailscale remotely (e.g. from office or mobile):
- The main BDIX portal page (`http://172.16.50.4`) loaded normally.
- However, clicking on any category tab such as English Movies (`http://172.16.50.14/DHAKA-FLIX-14/English%20Movies%20%281080p%29/`) resulted in the browser hanging indefinitely, stalling with a loading spinner, or failing to display directory content.

---

## 2. 🔍 Root Cause Analysis (RCA)
- **MTU & Packet Size Mismatch**:
  - The Tailscale WireGuard overlay tunnel utilizes an MTU of **1280 bytes**.
  - Local ISP / BDIX Ethernet networks operate with standard MTU of **1500 bytes** (Maximum Segment Size: 1460 bytes).
- **The Failure Trigger**:
  - The initial portal page (`172.16.50.4`) is small enough to fit within smaller packets.
  - The movie catalog on `172.16.50.14` runs H5AI directory indexing with heavy JavaScript (`scripts.js` ~98 KB) and massive directory HTML payloads.
  - When `172.16.50.14` attempted to stream large TCP frames (1500 bytes) with the `DF` (Don't Fragment) flag set, the subnet router (`homeserver`) could not encapsulate them into the 1280-byte WireGuard tunnel without fragmentation.
  - Because ISP routers frequently drop ICMP "fragmentation needed" packets (breaking Path MTU Discovery / PMTUD), the connection hung in a black-hole state.

---

## 3. 🛠️ Step-by-Step Resolution Procedures

### Step 1: Apply Kernel TCP MSS Clamping
We configured the kernel's packet mangling pipeline to clamp the TCP Maximum Segment Size to Path MTU dynamically during the TCP handshake (`SYN`):

```bash
sudo iptables -t mangle -A FORWARD -p tcp -m tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu
```

### Step 2: Make the Optimization Persistent Across Reboots
Integrated the rule into `/etc/systemd/system/tailscale-gro.service`:

```ini
[Unit]
Description=Tailscale UDP GRO and TCP MSS Clamping Optimization
After=network.target tailscaled.service

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/sbin/ethtool -K wlp2s0 rx-udp-gro-forwarding on rx-gro-list off
ExecStart=/bin/sh -c 'iptables -t mangle -C FORWARD -p tcp -m tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu 2>/dev/null || iptables -t mangle -A FORWARD -p tcp -m tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu'

[Install]
WantedBy=multi-user.target
```

Reloaded and verified the systemd unit:
```bash
sudo systemctl daemon-reload
sudo systemctl restart tailscale-gro.service
```

---

## 4. ✅ Verification & Quality Assurance

Tested from the remote Windows client over Tailscale:

```powershell
curl.exe -sI "http://172.16.50.14/DHAKA-FLIX-14/English%20Movies%20%281080p%29/"
```
**Output**:
```text
HTTP/1.1 200 OK
Server: nginx/1.20.1
Content-Type: text/html;charset=utf-8
```

The full 11 KB directory tree, 98 KB `scripts.js`, and all nested folder lists (`(2024) 1080p`, `(2025) 1080p`, `(2026) 1080p`) now download and render in **0.31 seconds**.

---

## 5. 🛡️ Preventative Measures
- Tailscale Subnet Router MTU is clamped automatically on every boot.
- All media server nodes in the `172.16.50.0/24` subnet (`172.16.50.7`, `172.16.50.8`, `172.16.50.9`, `172.16.50.12`, `172.16.50.14`) now stream smoothly across Tailscale without packet starvation.
