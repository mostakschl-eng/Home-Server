# 🐳 Container Ecosystem & Runtime Architecture

This document details the container runtime, Docker daemon specifications, network drivers, Traefik reverse proxy orchestration, and container inventory on `homeserver`.

---

## ⚙️ Docker Engine Specifications

```text
Docker Engine Version:    29.8.1 Community Edition
API Version:              1.56
Go Version:               go1.26.8
Git Commit:               4a63305
Container Runtime:        containerd://1.7.x (containerd.service active)
Storage Driver:           overlayfs
Default Address Pool:     10.0.0.0/8 (Subnet size /24 per network)
```

---

## 🗃️ Container Inventory & Health Status

| Container Name | Image Repository | Status | Host Port Mapping | Internal Role |
| :--- | :--- | :--- | :--- | :--- |
| **`coolify-proxy`** | `traefik:v3.6` | Up (healthy) | `80:80`, `443:443`, `8080:8080` | Ingress reverse proxy & SSL termination |
| **`coolify`** | `coollabsio/coolify:4.3.23` | Up (healthy) | `8000:8080` | Main PaaS backend & dashboard |
| **`coolify-realtime`** | `coollabsio/coolify-realtime:1.0.19` | Up (healthy) | `6001-6002:6001-6002` | Soketi WebSocket event engine |
| **`coolify-db`** | `postgres:15-alpine` | Up (healthy) | `None (Internal 5432)` | Relational database for Coolify state |
| **`coolify-redis`** | `redis:7-alpine` | Up (healthy) | `None (Internal 6379)` | In-memory key-value cache and queue |
| **`coolify-sentinel`** | `coollabsio/sentinel:1.0.1` | Up (healthy) | `None (Internal)` | Node resource & health telemetry agent |
| **`00lpnw9fxnqw...`** | `doobidoo/mcp-memory-service:latest` | Up | `8001:8000` | MCP AI Memory Service with ONNX embeddings |

---

## 🚦 Ingress & Reverse Proxy (Traefik v3.6)

Ingress routing is governed dynamically by `coolify-proxy` (Traefik v3.6). Traefik mounts `/var/run/docker.sock` in read-only mode to detect container creation, termination, and label updates without requiring proxy restarts.

### Traefik Operational Configuration
- **Entrypoints**:
  - `http`: Port 80 (configured with automatic 301 redirect to HTTPS)
  - `https`: Port 443 (TLS with SNI routing and HTTP/3 support)
  - `traefik`: Port 8080 (internal metrics and diagnostic dashboard)
- **Automatic Compression**: `gzip` and `zstd` compression middleware enabled.
- **Dynamic Routing**: Rules defined via Docker labels:
  ```yaml
  labels:
    - "traefik.enable=true"
    - "traefik.http.routers.http-0-app.entryPoints=http"
    - "traefik.http.routers.http-0-app.middlewares=gzip,redirect-to-https"
    - "traefik.http.routers.http-0-app.rule=Host(`service.domain.com`) && PathPrefix(`/`)"
    - "traefik.http.services.http-0-app.loadbalancer.server.port=8000"
  ```

---

## 🗄️ Docker Volumes & Mounts Matrix

| Volume / Host Directory | Bound Container | Target In-Container Mount | Purpose |
| :--- | :--- | :--- | :--- |
| `coolify-db` | `coolify-db` | `/var/lib/postgresql/data` | PostgreSQL persistent data files |
| `coolify-redis` | `coolify-redis` | `/data` | Redis persistent memory dump (`dump.rdb`) |
| `00lpnw...-mcp-memory-data` | `mcp-memory-service` | `/app/sqlite_db` | SQLite database storing AI memory vectors |
| `/data/coolify/source/.env` | `coolify` | `/var/www/html/.env` | Coolify environment secrets & configuration |
| `/data/coolify/proxy/` | `coolify-proxy` | `/traefik` | Traefik configuration files and certificates |
| `/data/coolify/sentinel/` | `coolify-sentinel` | `/app/db` | Sentinel database metrics store |
| `/var/run/docker.sock` | `coolify-proxy`, `sentinel` | `/var/run/docker.sock` | Docker socket API for dynamic discovery |
