# 🌐 Home Server Infrastructure Documentation

[![OS](https://img.shields.io/badge/OS-Ubuntu%2026.04.1%20LTS-E95420?logo=ubuntu&logoColor=white)](docs/01-system-specification.md)
[![Kernel](https://img.shields.io/badge/Kernel-Linux%207.0.0--31--generic-black?logo=linux&logoColor=white)](docs/01-system-specification.md)
[![Hardware](https://img.shields.io/badge/Hardware-HP%20EliteBook%20840%20G3-0096D6?logo=hp&logoColor=white)](docs/01-system-specification.md)
[![Docker](https://img.shields.io/badge/Docker-v29.8.1-2496ED?logo=docker&logoColor=white)](docs/05-container-ecosystem.md)
[![PaaS](https://img.shields.io/badge/PaaS-Coolify%20v4.3.23-6B46C1)](docs/06-deployed-services.md)
[![Network](https://img.shields.io/badge/Mesh-Tailscale%20Zero--Trust-black?logo=tailscale&logoColor=white)](docs/03-network-topology.md)

---

## 📌 Executive Summary

This repository hosts the **production-grade infrastructure documentation, architectural blueprints, runbooks, and incident history** for the personal home server (`homeserver`). 

Designed with enterprise standards inspired by hyperscaler documentation practices (Microsoft / Meta / Google Cloud), every aspect of the server—from physical hardware to network topologies, Docker container stacks, security posture, and automated maintenance—is systematically tracked and version-controlled.

---

## 🖥️ Server Quick Facts

| Dimension | Specification | Notes |
| :--- | :--- | :--- |
| **Hostname** | `homeserver` | Machine ID: `7a7aae592e0d4ddba2f5098c8f9eac25` |
| **Hardware Platform** | HP EliteBook 840 G3 | Laptop chassis with battery backup (built-in UPS) |
| **Processor (CPU)** | Intel Core i5-6300U @ 2.40GHz (3.00GHz Turbo) | 2 Cores, 4 Threads (Skylake architecture) |
| **Memory (RAM)** | 16 GB DDR4 (14.9 GiB usable) | ~1.4 GiB active load (~13 GiB free/cache) |
| **Storage (Primary SSD)** | 256 GB SanDisk SD7SN6S SATA SSD | LVM (`ubuntu-vg`), 231 GB root (6% used, 208 GB free) |
| **Operating System** | Ubuntu 26.04.1 LTS (*Resolute Raccoon*) | Linux Kernel 7.0.0-31-generic (x86_64) |
| **Local Network IP** | `192.168.0.149/24` (via `wlp2s0`) | DHCP assigned via primary gateway `192.168.0.1` |
| **Tailscale Mesh IP** | `100.81.129.68/32` (MagicDNS: `homeserver`) | Zero-Trust secure access from any location |
| **Container Engine** | Docker Community Edition 29.8.1 (API 1.56) | Orchestrated primarily via Coolify |
| **Application Platform** | Coolify v4.3.23 + Traefik v3.6 | Self-hosted PaaS managing web services & MCP agents |

---

## 🗂️ Documentation Hierarchy

```text
Home-Server/
│
├── README.md                          # Executive infrastructure summary & navigation
├── ARCHITECTURE.md                    # System architecture, container ingress & network topology
├── CHANGELOG.md                       # Historical log of upgrades and structural changes
│
├── requirements.txt                   # Python dependencies for automated SSH audits
│
├── docs/                              # Deep Technical Reference
│   ├── 01-system-specification.md     # OS, Kernel, Hardware, CPU/RAM, Power Management
│   ├── 02-storage-and-disks.md        # Storage hardware, LVM volumes, mounts, and health
│   ├── 03-network-topology.md         # Interfaces, Subnets, Routing, Tailscale, Port Matrix
│   ├── 04-security-and-access.md      # SSH hardening, Fail2ban, Firewall, User Privileges
│   ├── 05-container-ecosystem.md      # Docker daemon config, Docker networks, Traefik proxy
│   ├── 06-deployed-services.md        # Coolify suite, PostgreSQL, Redis, MCP Memory Service
│   ├── 07-maintenance-automation.md   # Weekly cleanup cron, Tailscale GRO service, log rotation
│   ├── 08-disaster-recovery.md        # Backups, SQLite snapshots, bare-metal restore SOP
│   └── 09-agent-developer-guide.md    # Operating guide & safety protocol for AI agents & engineers
│
├── runbooks/                          # Incident Post-Mortems & Operational Runbooks
│   ├── README.md                      # Runbook catalog & emergency troubleshooting guide
│   ├── TEMPLATE.md                    # Standard SOP & Incident Post-Mortem format
│   ├── inc-001-mcp-onnx-model-s3-timeout.md  # Fix: ONNX Model Injection into MCP Service
│   ├── inc-002-tailscale-udp-gro-optimization.md # Fix: Tailscale UDP GRO Systemd Service
│   └── inc-003-docker-disk-exhaustion-prevention.md # Fix: Automated Weekly Pruning Pipeline
│
└── scripts/                           # Operational Utilities
    └── server-audit.py                # Non-destructive inventory collection tool
```

---

## ⚡ Quick Operational Commands

### Remote Access via Tailscale
```bash
# Secure Shell connection
ssh mostak@100.81.129.68

# Or using MagicDNS (if Tailscale MagicDNS is enabled on client)
ssh mostak@homeserver
```

### Core Services Management
```bash
# Check running containers
sudo docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'

# View Coolify application logs
sudo docker logs -f coolify

# View Traefik proxy status
sudo docker logs -f coolify-proxy

# Monitor system resources
btop
# or
htop
```

### Health & Storage Checks
```bash
# Disk space summary
df -hT -x tmpfs -x squashfs

# Docker disk usage
sudo docker system df

# Memory utilization
free -h
```

---

## 🔒 Security Principles

1. **Zero-Trust Network Perimeter**: Access to management ports (Coolify Web UI, SSH, internal APIs) is routed exclusively over **Tailscale Mesh** or encrypted local LAN.
2. **Brute-Force Mitigation**: `fail2ban` protects the SSH daemon against authentication attacks.
3. **Automated Maintenance**: `/etc/cron.weekly/docker-cleanup` prevents container disk bloat.
4. **Isolated Service Mesh**: Services communicate over dedicated Docker bridge networks (`coolify`), with external routing handled exclusively via Traefik reverse proxy.
