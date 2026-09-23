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

### Security Recommendation
To lock down external LAN access while preserving Tailscale and Docker functionality:
1. Enforce traffic restriction using the `DOCKER-USER` chain or bind published ports to `127.0.0.1` and `100.81.129.68` rather than `0.0.0.0`.
2. Restrict non-Tailscale ingress if the server is exposed to untrusted networks.

---

## 🔄 Automated Security Patching

```text
Service Unit: unattended-upgrades.service (Active & running)
Configuration: /etc/apt/apt.conf.d/50unattended-upgrades
```
- **Automated Scope**: Security updates from `resolute-security` repository are installed unattended without requiring manual administrator intervention.
- **Kernel Updates**: Managed safely, with older kernels automatically pruned by `apt`.
