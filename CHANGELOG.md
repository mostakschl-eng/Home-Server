# 📜 Infrastructure Changelog

All notable technical updates, configuration modifications, and architectural milestones for `homeserver` are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [Unreleased]

### Fixed
- Resolved Nextcloud administrative setup warnings:
  - Configured `trusted_proxies` (`10.0.0.0/8`, `172.16.0.0/12`, `127.0.0.1`) and `forwarded_for_headers` (`HTTP_X_FORWARDED_FOR`) to properly recognize Traefik reverse proxy and resolve client IP spoofing vulnerability.
  - Performed expensive database mimetype repair (`occ maintenance:repair --include-expensive`).
  - Configured off-peak background maintenance window (`maintenance_window_start = 20` UTC / 2:00 AM BST).
  - Configured default phone validation region to `BD` (`default_phone_region = BD`).
  - Configured valid server identifier integer (`serverid = 1`).
  - Disabled unused containerized AppAPI daemon (`app_api`) to prevent CPU and memory exhaustion on the dual-core host.
  - Archived historical setup/token log errors into `nextcloud.log.bak`.
- Added automatic `.env` loading to `scripts/server-audit.py`.

### Changed
- Reduced live Coolify build concurrency from two to one and trimmed Preview Generator's background size specifications from 12 to five. Added a privacy-safe, rotating upload timing log in the Nextcloud application nginx configuration. All values, generated specifications, nginx syntax, and service health were verified.
- Tuned live Nextcloud image processing: one concurrent new preview, a 2048-pixel maximum on each axis, JPEG preview quality 80, and a 60-second Preview Generator job budget every five minutes. Switched transactional file locking from APCu to the existing Redis container while keeping local APCu caching. Settings, Redis implementation, and status endpoint were verified; upload CPU reduction has not been measured.
- Replaced the obsolete once-per-minute `Downloads` scan with a five-minute `Media Downloads` scan protected by `flock`; a manual scan completed with zero errors. See `runbooks/inc-008-nextcloud-upload-cpu-tuning.md` for backups, verification, rollback, and the read-only Coolify review.
- Clarified that agent rules are procedural rather than OS-enforced, and documented the administrative account's Docker privilege.
- Removed password-in-command SSH examples; the read-only audit now rejects unverified host keys.
- Updated the Aria2 downloader guide for the live `Media Downloads` storage mount and documented that the Nextcloud scan cron still targets the old `Downloads` folder.

### Added
- Added a local-only Go media optimizer for lossless JPEG/PNG/WebP processing, video stream-copy metadata stripping, and resumable Nextcloud queueing. It has not been deployed or connected to the server.
- Consolidated the local test bench into `scripts/media-optimizer/`, preserving the seven original samples and removing generated outputs and bundled tools; added workflow and future server setup documentation.

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
- Fixed Coolify and Nextcloud site inaccessibility (`404 page not found` / crash on boot) by configuring persistent kernel non-local IP binding (`net.ipv4.ip_nonlocal_bind=1`), recreating Coolify stack, and bridging Traefik reverse proxy to the Nextcloud service network (`INC-007`).

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
- Added privacy-safe PHP request CPU/timing logs and recorded the latest 21-image upload diagnosis; core health-check apply remains blocked by the required Tier 2 confirmation.
- Simplified agent approvals at operator request: routine reversible tuning uses existing authorization, no exact confirmation token; destructive and access-changing work retains risk review and tool approval boundaries.
- Fixed Media Downloads cron argument handling using direct PHP occ; verified the recent movie in Nextcloud file cache and added the home-router Tailscale setup guide.
- Set Nextcloud preview JPEG quality to 70 for the requested visual trial; verified saved configuration and healthy status. Existing cached previews and original files retained.
- Excluded Aria2 resume sidecars from Nextcloud indexing, refreshed the existing auxiliary cache entry, and verified media visibility without deleting movie or resume data.
- Changed Nextcloud preview JPEG quality from 70 to 60 for the authorized visual/loading trial; verified saved value and healthy status.
- Audited permanent deletion and ran native orphan-preview cleanup; verified orphan preview entries decreased from 564 across 68 missing sources to zero.
- Completed requested native preview-cache reset and started low-priority sequential regeneration at JPEG quality 60; original files preserved.
- Verified full preview regeneration completed: 468 source IDs and 2787 cached preview entries at the current quality/sizing configuration.
