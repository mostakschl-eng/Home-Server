# 🏗️ System Architecture & Ingress Topology

This document details the architectural layout, ingress routing, network mesh, and storage model of the `homeserver` environment.

---

## 🗺️ High-Level Infrastructure Topology

```mermaid
graph TD
    subgraph WAN ["🌐 External Network / Remote Clients"]
        ClientDesktop["💻 Desktop (desktop-vpvd0md)<br/>100.104.22.76"]
        ClientPhone["📱 Galaxy S22 Ultra<br/>100.112.83.33"]
        ClientUser["💻 User Station<br/>100.104.142.55"]
    end

    subgraph Mesh ["🔒 Tailscale Encrypted WireGuard Mesh (100.64.0.0/10)"]
        TailscaleNode["homeserver (Tailscale Node)<br/>100.81.129.68:41641 UDP"]
    end

    subgraph Host ["🖥️ Physical Host: HP EliteBook 840 G3 (Ubuntu 26.04 LTS)"]
        HostNetwork["Interface: wlp2s0 (192.168.0.149)<br/>Interface: tailscale0 (100.81.129.68)"]
        SSHService["OpenSSH Server (sshd)<br/>Port: 22 TCP (Fail2Ban Protected)"]
        TailscaleGRO["tailscale-gro.service<br/>ethtool UDP GRO Forwarding"]
        
        subgraph DockerDaemon ["🐳 Docker Engine 29.8.1"]
            subgraph Ingress ["Ingress & Reverse Proxy"]
                TraefikProxy["coolify-proxy (Traefik v3.6)<br/>Ports: 80, 443, 8080 TCP/UDP"]
            end

            subgraph CoolifyCore ["Coolify PaaS Infrastructure"]
                CoolifyApp["coolify (Laravel Web/API)<br/>Port: 8000 (Internal 8080)"]
                CoolifyRealtime["coolify-realtime (Soketi WebSockets)<br/>Ports: 6001, 6002"]
                CoolifyDB["coolify-db (PostgreSQL 15)<br/>Internal Port: 5432"]
                CoolifyRedis["coolify-redis (Redis 7)<br/>Internal Port: 6379"]
                CoolifySentinel["coolify-sentinel (1.0.1)<br/>System & Health Telemetry"]
            end

            subgraph AppTier ["User Applications Tier"]
                MCPMemory["mcp-memory-service (AI Context)<br/>Port: 8001 (Internal 8000)<br/>Domain: *.sslip.io"]
            end
        end

        subgraph StorageLayer ["💾 Storage & Volumes"]
            DiskRoot["SSD Root Partition: /dev/sda3 (LVM)<br/>Size: 231 GB ext4"]
            CoolifyDataDir["/data/coolify/<br/>Configs, Source, Proxy rules, SSH keys"]
            DockerVolumes["/var/lib/docker/volumes/<br/>coolify-db, coolify-redis, mcp-memory-data"]
        end
    end

    ClientDesktop -->|Encrypted WireGuard| TailscaleNode
    ClientPhone -->|Encrypted WireGuard| TailscaleNode
    ClientUser -->|Encrypted WireGuard| TailscaleNode

    TailscaleNode --> HostNetwork
    HostNetwork --> SSHService
    HostNetwork --> TraefikProxy
    HostNetwork --> CoolifyApp

    TraefikProxy -->|Bridge: coolify 10.0.1.0/24| CoolifyApp
    TraefikProxy -->|Bridge: coolify 10.0.1.0/24| CoolifyRealtime
    TraefikProxy -->|Bridge: coolify 10.0.1.0/24| MCPMemory

    CoolifyApp --> CoolifyDB
    CoolifyApp --> CoolifyRedis
    CoolifyApp --> CoolifySentinel
    CoolifySentinel -->|/var/run/docker.sock| DockerDaemon

    CoolifyDB --> DockerVolumes
    CoolifyRedis --> DockerVolumes
    MCPMemory --> DockerVolumes
    CoolifyApp --> CoolifyDataDir
```

---

## 🚦 Ingress Traffic Flow & Port Mapping

| Port (Host) | Protocol | Component | Target Container | Routing Policy / Domain |
| :--- | :--- | :--- | :--- | :--- |
| **22** | TCP | Host OpenSSH | Native (`sshd`) | Direct SSH with Fail2Ban protection |
| **80** | TCP | Traefik Reverse Proxy | `coolify-proxy` | HTTP Ingress / Auto-redirect to HTTPS |
| **443** | TCP / UDP | Traefik Reverse Proxy | `coolify-proxy` | HTTPS Ingress / HTTP/3 (QUIC) support |
| **6001 - 6002** | TCP | Soketi WebSockets | `coolify-realtime` | Real-time event broadcasting for Coolify |
| **8000** | TCP | Coolify Management UI | `coolify` | Main web management dashboard |
| **8001** | TCP | MCP Memory Service | `00lpnw9fxnqwf6f8lzonywhk` | Direct TCP fallback / API ingress |
| **8080** | TCP | Traefik Dashboard | `coolify-proxy` | Traefik diagnostic endpoint |
| **41641** | UDP | Tailscale WireGuard | Native (`tailscaled`) | Direct WireGuard transport communication |

---

## 🌐 Docker Networking Architecture

Docker manages an isolated bridge network dedicated to Coolify services:

```text
Bridge Name:       br-9c58423ac515
Docker Network:    coolify
Subnet:            10.0.1.0/24
IPv6 Subnet:       fd92:5548:7b6::/64
Gateway:           10.0.1.1
Attachable:        true
Scope:             local
```

### Communication Principles
1. **Container Isolation**: User services inside the `coolify` network communicate via container names or aliases (e.g. `coolify-db`, `coolify-redis`).
2. **Reverse Proxy Routing**: Incoming requests on Ports 80 and 443 are evaluated by Traefik dynamically using Docker labels (`traefik.enable=true`, `traefik.http.routers.*`).
3. **Internal Only DBs**: Databases (`coolify-db` on port 5432, `coolify-redis` on port 6379) are not published to host interfaces; they are only accessible to authorized containers within the bridge network.

---

## 💾 Storage Architecture & Persistence

```text
/dev/sda (256 GB SanDisk SSD)
├── sda1 (1.0 GB)  --> /boot/efi (FAT32, UEFI boot files)
├── sda2 (2.0 GB)  --> /boot (ext4, Kernel images & initramfs)
└── sda3 (235.4 GB) --> LVM Volume Group: ubuntu-vg
    └── ubuntu-lv (231 GB) --> Mountpoint: / (ext4)
        ├── /swap.img (4.0 GB swapfile)
        ├── /data/coolify/ (PaaS configurations, proxy rules, .env files)
        └── /var/lib/docker/volumes/
            ├── coolify-db (PostgreSQL database state)
            ├── coolify-redis (Redis in-memory snapshot state)
            └── 00lpnw9fxnqwf6f8lzonywhk-mcp-memory-data (SQLite vector / context database)
```
