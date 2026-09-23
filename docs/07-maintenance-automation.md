# ⚙️ Maintenance & System Automation

This document outlines the scheduled maintenance routines, custom automation scripts, and background systemd units configured on `homeserver` to ensure high availability and zero administrative overhead.

---

## 📅 Scheduled Tasks Matrix

| Schedule | Task / Service | Target Location | Objective |
| :--- | :--- | :--- | :--- |
| **Weekly** | Docker Cleanup | `/etc/cron.weekly/docker-cleanup` | Prune build caches, dangling images, old containers, and APT cache |
| **Boot / Network** | Tailscale GRO | `/etc/systemd/system/tailscale-gro.service` | Enable kernel UDP receive offload for high-bandwidth WireGuard traffic |
| **Daily** | Logrotate | `/etc/cron.daily/logrotate` | Compress and rotate system logs to prevent disk saturation |
| **Daily** | APT Compatibility | `/etc/cron.daily/apt-compat` | Refresh repository package catalogs for security checks |
| **Daily** | DPKG & ManDB | `/etc/cron.daily/dpkg`, `man-db` | Database maintenance |

---

## 🧹 Weekly Docker & System Cleanup Script

Located at `/etc/cron.weekly/docker-cleanup`, this script runs automatically with root privileges once per week:

```bash
#!/bin/sh
# 1. Prune dangling build cache older than 48 hours
docker builder prune -a --filter "until=48h" -f >/dev/null 2>&1

# 2. Prune dangling (untagged) images
docker image prune -f >/dev/null 2>&1

# 3. For each repository, retain only the 3 newest images and remove older iterations
for repo in $(docker images --format "{{.Repository}}" | sort -u | grep -v "<none>"); do
  docker images "$repo" -q | awk 'NR>3' | xargs -r docker rmi >/dev/null 2>&1
done

# 4. Remove stopped containers older than 7 days (168 hours)
docker container prune -f --filter "until=168h" >/dev/null 2>&1

# 5. Clean APT package manager cache
apt-get clean >/dev/null 2>&1
```

### Operational Benefits
- Prevents Docker's storage driver (`overlayfs`) from accumulating gigabytes of historical build layers.
- Guarantees that active application rollbacks remain possible (retains up to 3 previous image tags) while eliminating obsolete iterations.
- Frees space in `/var/cache/apt/archives/`.

---

## 🚀 Tailscale UDP GRO Systemd Unit

Located at `/etc/systemd/system/tailscale-gro.service`:

```ini
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

### Why These Are Essential
1. **Generic Receive Offload (`rx-udp-gro-forwarding on`)**: Coalesces incoming WireGuard UDP packets into larger frames, reducing CPU overhead by up to **60%** and maximizing mesh throughput.
2. **TCP MSS Clamping (`TCPMSS --clamp-mss-to-pmtu`)**: Prevents remote subnet responses (such as BDIX media servers) from exceeding the Tailscale tunnel's 1280-byte MTU (`INC-005`).
3. **ISP TTL Normalization (`TTL --ttl-set 64`)**: Overcomes upstream ISP anti-tethering / anti-routing filters that silently drop container forwarded outbound packets (`TTL=63`), enabling all Docker services to reach public WAN APIs (`INC-006`).

---

## 🔍 Verification & Health Commands

Run these commands periodically or during health audits:

```bash
# Verify tailscale-gro service status
systemctl status tailscale-gro.service

# Manually test the weekly docker cleanup pipeline
sudo /etc/cron.weekly/docker-cleanup

# Check Docker disk usage before and after cleanup
sudo docker system df

# Check systemd failed services
systemctl --failed
```
