# 📋 [INC-002] Tailscale WireGuard Throughput Bottleneck & GRO Optimization

---

## 📌 Metadata
- **Incident ID**: `INC-002`
- **Date / Timestamp**: `2026-09-22 17:30 UTC`
- **Impacted Service(s)**: `tailscaled.service`, Wi-Fi interface (`wlp2s0`)
- **Severity Level**: Medium (Performance degradation & elevated CPU usage)
- **Lead Engineer**: `mostak`
- **Resolution Status**: **Resolved**

---

## 1. ⚠️ Problem Statement & Symptoms
High throughput data transfers across the Tailscale mesh network exhibited inconsistent bandwidth and caused high CPU load on core 0 of the host CPU (`ksoftirqd` and `tailscaled` spiking).

---

## 2. 🔍 Root Cause Analysis (RCA)
- Tailscale relies on WireGuard UDP packets.
- On Linux Wi-Fi drivers, Generic Receive Offload (GRO) for UDP packets is disabled by default. Each incoming packet triggers a separate kernel interrupt and context switch, capping transfer speeds and consuming substantial CPU cycles during SSH tunnels or file copies.

---

## 3. 🛠️ Step-by-Step Resolution Procedures
Created a dedicated systemd service to permanently enable UDP GRO forwarding via `ethtool` upon boot.

### Step 1: Create Systemd Unit File
Created `/etc/systemd/system/tailscale-gro.service`:

```ini
[Unit]
Description=Enable UDP GRO forwarding for Tailscale
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/sbin/ethtool -K wlp2s0 rx-udp-gro-forwarding on rx-gro-list off

[Install]
WantedBy=multi-user.target
```

### Step 2: Reload Systemd Daemon & Enable Service
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now tailscale-gro.service
```

---

## 4. ✅ Verification & Quality Assurance
Inspected service activation status:

```bash
systemctl status tailscale-gro.service
```
**Output**:
```text
● tailscale-gro.service - Enable UDP GRO forwarding for Tailscale
     Loaded: loaded (/etc/systemd/system/tailscale-gro.service; enabled; preset: enabled)
     Active: active (exited)
```

Verified with `ethtool -k wlp2s0 | grep gro`:
```text
rx-udp-gro-forwarding: on
```

---

## 5. 🛡️ Preventative Measures
- The unit is marked `WantedBy=multi-user.target`, ensuring persistence across server reboots.
