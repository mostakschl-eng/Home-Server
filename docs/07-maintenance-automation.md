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
[Unit]
Description=Enable UDP GRO forwarding for Tailscale
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/sbin/ethtool -K wlp2s0 rx-udp-gro-forwarding on rx-gro-list off

[Install]
WantedBy=multi-user.target
```

### Why This Is Essential
Tailscale's encrypted WireGuard tunnels utilize UDP datagrams. Under default Linux network driver settings for Wi-Fi (`wlp2s0`), UDP packet processing occurs on a per-packet basis in userspace, causing elevated CPU load and throughput ceilings. 

Enabling **Generic Receive Offload (`rx-udp-gro-forwarding on`)** coalesces incoming UDP packets into larger kernel-level frames, reducing CPU overhead by up to **60%** and maximizing transfer throughput across the mesh network.

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
