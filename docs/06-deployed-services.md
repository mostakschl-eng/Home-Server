# 🚀 Deployed Services Catalog

This document details all applications and background services deployed on `homeserver`, including operational URLs, volume structures, and configuration manifests.

---

## 1. 🎛️ Coolify PaaS (Platform as a Service)

Coolify is an open-source, self-hosted Heroku / Netlify / Vercel alternative providing unified deployment workflows, reverse-proxy management, and database provisioning.

```text
Version:          4.3.23
Dashboard URL:    http://100.81.129.68:8000 (Direct Tailscale Port)
Proxy Domain:     http://coolify.100.81.129.68.sslip.io (Traefik Ingress)
Access Policy:    LOCKED on Home Wi-Fi (192.168.0.149:8000 -> Connection Refused)
Config Path:      /data/coolify/source/
Environment File: /data/coolify/source/.env
```

### Component Roles
- **Web App / API Engine** (`coollabsio/coolify:4.3.23`):
  - Laravel-based orchestrator handling Git webhooks, deployments, Docker Compose parsing, and server resource monitoring.
- **Realtime Gateway** (`coollabsio/coolify-realtime:1.0.19`):
  - Soketi-based WebSocket engine providing instant terminal output, log streaming, and deployment progress updates to the web interface.
  - Host Ports: `6001` (client WebSockets), `6002` (management API).
- **PostgreSQL Database** (`postgres:15-alpine`):
  - Stores all server definitions, project trees, environment configurations, and deployment logs.
  - Volume: `coolify-db`.
- **Redis Cache & Queues** (`redis:7-alpine`):
  - Asynchronous background worker queue for deployment tasks and real-time cache.
  - Volume: `coolify-redis`.
- **Sentinel Agent** (`coollabsio/sentinel:1.0.1`):
  - Daemon container reporting CPU, RAM, and disk utilization statistics to the Coolify dashboard.

---

## 2. 🧠 MCP Memory Service (`mcp-memory-service`)

A high-performance persistent context and memory layer implementing the **Model Context Protocol (MCP)**, allowing AI coding assistants and agents to store and recall long-term project knowledge and conversation memory.

```text
Container Name:   00lpnw9fxnqwf6f8lzonywhk-081811531731
Base Image:       doobidoo/mcp-memory-service:latest
Host Direct Port: 100.81.129.68:8001 -> 8000 TCP (Tailscale Only, Locked from Local Wi-Fi)
Proxy Routing:    Traefik reverse proxy
Dynamic Domain:   00lpnw9fxnqwf6f8lzonywhk.223.29.215.54.sslip.io
Project Name:     ai-memory
Environment:      production
```

### Storage & Persistence
- **Vector / SQLite DB**: `00lpnw9fxnqwf6f8lzonywhk-mcp-memory-data` -> `/app/sqlite_db`
- **Backups Volume**: Bound to persistent volume for database dump retention.

### Embedding Model Acceleration
- Utilizes the **`all-MiniLM-L6-v2`** ONNX sentence transformer model to generate dense semantic vector embeddings.
- Model weights are pre-cached inside the container filesystem under:
  `/root/.cache/mcp_memory/onnx_models/all-MiniLM-L6-v2/`
  *(Preventing cold-start latency and external S3 download dependency).*

---

## 3. ☁️ Nextcloud Private Cloud Suite

Nextcloud Hub 35 (Linuxserver edition) deployed via Coolify providing self-hosted personal cloud storage, document sync, WebDAV, and automatic photo backup for mobile and desktop clients.

```text
Service Name:     nextcloud-syo7laqurhdbmmidrumfckbb
Application URL:  http://nextcloud.100.81.129.68.sslip.io
Routing Policy:   Traefik Reverse Proxy (Internal Port 80)
Project Name:     personal-storage
Environment:      production
```

### Component Architecture
- **Nextcloud Web Engine** (`lscr.io/linuxserver/nextcloud:latest`):
  - PHP 8.x + Nginx + APCu memory caching and transactional file locking.
  - Volumes:
    - `/config`: `syo7laqurhdbmmidrumfckbb_nextcloud-config`
    - `/data`: `syo7laqurhdbmmidrumfckbb_nextcloud-data`
- **PostgreSQL 16 Database** (`postgres:16-alpine`):
  - Dedicated relational database holding 348 Nextcloud system and user tables.
  - Container Name: `nextcloud-db-syo7laqurhdbmmidrumfckbb`
  - Volume: `syo7laqurhdbmmidrumfckbb_nextcloud-postgresql-data`
- **Redis 7.4 Cache** (`redis:7.4-alpine`):
  - In-memory cache and transactional lock queue for rapid WebDAV synchronization.
  - Container Name: `redis-syo7laqurhdbmmidrumfckbb`
  - Volume: `syo7laqurhdbmmidrumfckbb_nextcloud-redis-data`

---

## 4. 🛡️ System Host Services

| Service | Daemon | Function |
| :--- | :--- | :--- |
| **OpenSSH** | `sshd.service` | Encrypted administrative remote terminal access |
| **Tailscale** | `tailscaled.service` | Point-to-point WireGuard mesh network client |
| **Tailscale GRO** | `tailscale-gro.service` | Custom kernel UDP Generic Receive Offload optimization |
| **Fail2Ban** | `fail2ban.service` | Log monitor banning rogue SSH authentication attempts |
| **Chrony** | `chrony.service` | Precision NTP network time protocol synchronizer |
| **Thermal Daemon** | `thermald.service` | Laptop chassis thermal regulation and cooling policies |
| **Unattended Upgrades** | `unattended-upgrades.service` | Automated application of critical security patches |
