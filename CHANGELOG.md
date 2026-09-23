# 📜 Infrastructure Changelog

All notable technical updates, configuration modifications, and architectural milestones for `homeserver` are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [Unreleased]

### Planned
- [ ] Migrate primary network uplink from Wi-Fi (`wlp2s0`) to Gigabit Ethernet (`enp0s31f6`) for lower latency and reduced packet jitter.
- [ ] Enforce SSH public-key-only authentication (disable password logins).
- [ ] Implement automated off-site backups for `/data/coolify` and Docker volumes to external cloud storage (S3 / R2 / Backblaze).
- [ ] Configure Prometheus + Grafana telemetry dashboard for node metrics.

---

## [2026-09-23] - Enterprise Documentation & Repository Initialization

### Added
- Initialized official Git repository for `homeserver` infrastructure documentation (`mostakschl-eng/Home-Server`).
- Standardized documentation structure (`/docs`, `/runbooks`, `/scripts`).
- Created non-destructive automated server inventory audit script (`scripts/server-audit.py`).
- Added incident runbooks for past operational events:
  - `inc-001-mcp-onnx-model-s3-timeout.md`
  - `inc-002-tailscale-udp-gro-optimization.md`
  - `inc-003-docker-disk-exhaustion-prevention.md`
  - `inc-004-safe-firewall-and-port-isolation.md`
  - `inc-005-bdix-subnet-mss-clamping.md`
  - `inc-006-docker-isp-ttl-drop.md`
- Documented full system architecture, network topology, and ingress maps in `ARCHITECTURE.md`.
- Added `requirements.txt` defining Python dependencies (`paramiko`, `cryptography`, `bcrypt`) for headless SSH telemetry.
- Created `docs/09-agent-developer-guide.md` specifying rules of engagement, credential setups, and safety guidelines for future AI agents and engineers.
- Added `runbooks/inc-004-safe-firewall-and-port-isolation.md` documenting dual-network SSH access (LAN + Tailscale) and zero-lockout UFW hardening.
- Deployed Nextcloud Hub 35 Private Cloud Suite (`nextcloud`, `postgres:16-alpine`, `redis:7.4-alpine`) with Traefik routing at `http://nextcloud.100.81.129.68.sslip.io`.

### Fixed
- Added user `mostak` to the `docker` system group, enabling seamless non-sudo Docker CLI access for administrators and automated agents.
- Optimized Intel Wi-Fi power scheme via `/etc/modprobe.d/iwlmvm.conf` (`power_scheme=1`) to eliminate power-save throttling and packet jitter.
- Automated MCP Memory container ONNX model verification via `/home/mostak/onnx_model/ensure_model.sh` and `@reboot` cron job, preventing startup crash loops upon container recreation.
- Enforced Zero-Trust Tailscale port binding for Coolify (`100.81.129.68:8000`) and MCP Memory (`100.81.129.68:8001`), locking out unauthorized access from Local Home Wi-Fi while preserving dual-path SSH (Port 22) for maintenance.
- Resolved BDIX Subnet media server (`172.16.50.14`) tab loading freeze by enabling kernel TCP MSS Clamping to prevent MTU packet drops across the Tailscale WireGuard tunnel.
- Fixed container outbound WAN internet drops (`Could not fetch list of apps from the App Store`) by configuring iptables TTL normalization (`TTL --ttl-set 64`) to defeat upstream ISP anti-tethering filters (`INC-006`).

---

## [2026-09-22] - Container Maintenance & Optimization

### Added
- Implemented `/etc/cron.weekly/docker-cleanup` script:
  - Prunes dangling build cache older than 48 hours (`docker builder prune -a`).
  - Prunes untagged images.
  - Retains only the 3 latest images per repository tag, removing older iterations.
  - Removes stopped containers older than 7 days (`docker container prune`).
  - Runs `apt-get clean` to prevent package archive accumulation.
- Configured custom systemd unit `/etc/systemd/system/tailscale-gro.service`:
  - Executes `/usr/sbin/ethtool -K wlp2s0 rx-udp-gro-forwarding on rx-gro-list off`.
  - Dramatically improves Tailscale WireGuard throughput and CPU offloading over Wi-Fi.

### Deployed
- Deployed `mcp-memory-service` (`doobidoo/mcp-memory-service:latest`) under Coolify:
  - Container port 8000 exposed via host port 8001.
  - Persistent volume attached: `00lpnw9fxnqwf6f8lzonywhk-mcp-memory-data:/app/sqlite_db`.
  - Embedded local ONNX transformer model (`all-MiniLM-L6-v2`) in `/root/.cache/mcp_memory/`.

---

## [2026-09-21] - Platform Setup & OS Provisioning

### Added
- Installed Ubuntu 26.04.1 LTS (*Resolute Raccoon*) with Linux Kernel 7.0.0-31-generic on HP EliteBook 840 G3.
- Configured LVM storage partition scheme:
  - 1 GB EFI partition (`/boot/efi`).
  - 2 GB Boot partition (`/boot`).
  - 235.4 GB Volume Group (`ubuntu-vg`) with 231 GB root ext4 logical volume.
  - 4 GB swapfile (`/swap.img`).
- Installed Docker Engine Community Edition v29.8.1 (API 1.56).
- Deployed Coolify v4.3.23 self-hosted PaaS platform:
  - Traefik v3.6 reverse proxy & SSL manager.
  - PostgreSQL 15 database container.
  - Redis 7 cache container.
  - Soketi realtime WebSocket service.
  - Coolify Sentinel telemetry collector.
- Installed and connected Tailscale Mesh (`homeserver` @ `100.81.129.68`).
- Enabled `fail2ban` service for SSH brute-force defense.
- Verified system hardware: Intel Core i5-6300U, 16 GB DDR4 RAM, 256 GB SanDisk SSD.
