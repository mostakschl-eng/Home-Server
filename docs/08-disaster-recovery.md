# 🚨 Disaster Recovery & Backup Procedures

This document outlines recovery targets, backup procedures, bare-metal re-provisioning steps, and emergency access protocols for `homeserver`.

---

## 🎯 Recovery Objectives

| Metric | Target | Rationale |
| :--- | :--- | :--- |
| **Recovery Point Objective (RPO)** | 24 Hours | Databases and environment files should not lose more than 1 day of data. |
| **Recovery Time Objective (RTO)** | < 1 Hour | Rapid redeployment of Docker engine and restoration of volumes from cold backups. |

---

## 💾 Mission-Critical Data Artifacts

To fully reconstruct the server, only three data groups must be preserved:

```text
Critical Artifact                       Location on Server                               Target Format
──────────────────────────────────────────────────────────────────────────────────────────────────────
1. Coolify State & Environments        /data/coolify/source/.env                         Plain text / Secret vault
2. Coolify System Database              Docker Volume: coolify-db                         pg_dump SQL / Volume tar
3. AI Memory Context SQLite DB          Docker Volume: ...-mcp-memory-data                SQLite binary / .db dump
```

---

## 📦 Backup Procedures

### 1. Coolify PostgreSQL Database Backup
To extract a consistent SQL snapshot of all Coolify configuration and project definitions:

```bash
# Execute pg_dump directly inside the postgres container
sudo docker exec coolify-db pg_dump -U coolify -d coolify > ~/coolify_db_backup_$(date +%F).sql

# Compress snapshot
gzip ~/coolify_db_backup_$(date +%F).sql
```

### 2. MCP AI Memory Database Backup
To safely snapshot the live SQLite database without locking write transactions:

```bash
# Run safe SQLite online backup
sudo docker exec 00lpnw9fxnqwf6f8lzonywhk-081811531731 \
  sqlite3 /app/sqlite_db/memory.db ".backup '/app/backups/memory_backup_$(date +%F).db'"
```

### 3. Tar Archive of `/data/coolify` Configurations
```bash
sudo tar -czvf ~/coolify_configs_$(date +%F).tar.gz /data/coolify/
```

---

## 🚑 Emergency Access Protocols

### Scenario A: Tailscale Mesh is Unreachable
If the Tailscale daemon fails or internet connectivity drops:
1. Connect directly to the local home Wi-Fi network.
2. Locate the server via its LAN IPv4:
   ```bash
   ssh mostak@192.168.0.149
   ```
3. Check and restart Tailscale:
   ```bash
   sudo systemctl restart tailscaled
   tailscale status
   ```

### Scenario B: Wi-Fi Interface Disconnected / Misconfigured
If Wi-Fi (`wlp2s0`) loses carrier or credentials:
1. Connect a physical RJ45 Ethernet patch cable to the Gigabit port (`enp0s31f6`).
2. The network interface will negotiate link speed and request a DHCP lease automatically via `netplan` and `systemd-networkd`.
3. Locate the new IP on your router dashboard or scan with `nmap -sn 192.168.0.0/24`.

---

## ♻️ Bare-Metal Recovery Checklist

In the event of physical drive replacement or complete OS reinstallation:
1. **OS Installation**: Install Ubuntu 26.04 LTS (UEFI mode, standard LVM partition scheme).
2. **User Creation**: Set up administrative user `mostak` with sudo privileges.
3. **Core Dependencies**:
   ```bash
   sudo apt-get update && sudo apt-get install -y curl git ethtool fail2ban chrony
   ```
4. **Tailscale Deployment**:
   ```bash
   curl -fsSL https://tailscale.com/install.sh | sh
   sudo tailscale up
   ```
5. **Docker Engine Installation**:
   ```bash
   curl -fsSL https://get.docker.com | sh
   ```
6. **Restore Coolify & Volumes**:
   - Extract `/data/coolify/` archive.
   - Run Coolify automated installer script:
     ```bash
     curl -fsSL https://cdn.coollabs.io/coolify/install.sh | bash
     ```
   - Restore database dumps into `coolify-db` and MCP Memory volume.
