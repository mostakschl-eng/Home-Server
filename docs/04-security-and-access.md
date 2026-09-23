# 🛡️ Security Posture & Access Control

This document details the security posture, authentication protocols, intrusion prevention measures, user privileges, and firewall architecture of `homeserver`.

---

## 👤 User & Privilege Hierarchy

The server implements a least-privilege administrative model:

```text
User:         mostak
UID:          1000
Primary GID:  1000 (mostak)
Groups:       mostak, adm, cdrom, sudo, dip, plugdev, users, lxd
Home:         /home/mostak
Shell:        /bin/bash
```

### Administrative Privileges (`sudo`)
- The user `mostak` has standard `sudo` privileges configured via `/etc/sudoers` (`%sudo ALL=(ALL:ALL) ALL`).
- Direct root login is disabled for interactive SSH sessions by default.

---

## 🔑 Remote Access & SSH Hardening

SSH is provided by **OpenBSD Secure Shell (`sshd`)** listening on default port `22`.

```text
Service Unit:        ssh.service (Active & running)
Host Keys:           ED25519, RSA, ECDSA
Authorized Keys:     /home/mostak/.ssh/authorized_keys
Primary Access:      Tailscale Mesh IP (100.81.129.68) or Local Subnet (192.168.0.149)
```

### Fail2Ban Intrusion Prevention
The server runs `fail2ban.service` actively:
- **Jails Monitored**: `sshd`
- **Mechanism**: Inspects `/var/log/auth.log` for continuous authentication failures and dynamically injects iptables drop rules to ban offending external IP addresses.

---

## 🧱 Firewall Posture & Docker Packet Filtering

### Current Status
```text
UFW Status: Inactive
```

### Technical Rationale & Docker Interplay
In containerized environments running Docker Engine, Docker automatically manipulates `iptables` directly:
- Creates the `DOCKER`, `DOCKER-USER`, and `DOCKER-ISOLATION` packet filtering chains.
- Routes incoming external traffic on published container ports directly to container private IPs via NAT (Network Address Translation).
- Enabling naive UFW without Docker-aware rules often results in Docker bypassing standard UFW restrictions due to NAT packet priority.

### Active Zero-Trust Port Isolation Architecture
To lock down external Wi-Fi LAN access while guaranteeing zero-risk remote access:
1. **Dual-Path Emergency SSH Access**: Port 22 (`sshd`) remains bound to `0.0.0.0:22` (Fail2Ban protected), allowing terminal logins from both Local Wi-Fi (`192.168.0.149`) and Tailscale (`100.81.129.68`).
2. **Tailscale-Only Application Binding**: Management services are explicitly bound to the Tailscale IP (`100.81.129.68`), preventing Docker from listening on Wi-Fi:
   - Coolify Dashboard: `100.81.129.68:8000` (Local Wi-Fi returns *Connection refused*)
   - MCP Memory Service: `100.81.129.68:8001` (Local Wi-Fi returns *Connection refused*)
3. **Ghost Mode on Home LAN**: Any rogue device or guest connected to the home Wi-Fi scanning the server IP cannot detect or access Coolify or internal AI APIs.

---

## 🔄 Automated Security Patching

```text
Service Unit: unattended-upgrades.service (Active & running)
Configuration: /etc/apt/apt.conf.d/50unattended-upgrades
```
- **Automated Scope**: Security updates from `resolute-security` repository are installed unattended without requiring manual administrator intervention.
- **Kernel Updates**: Managed safely, with older kernels automatically pruned by `apt`.
