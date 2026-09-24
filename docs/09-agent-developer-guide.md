# 🤖 AI Agent & Developer Operating Guide

This guide establishes standard operating procedures for **AI coding agents (e.g. Antigravity, Claude, Gemini)** and software engineers interacting with the `homeserver` node. 

Follow this protocol to inspect, debug, manage, or deploy services safely without trial-and-error or risking downtime.

---

## ⚡ Quickstart for AI Agents

To work with this server, install the required automation dependencies:

```bash
# 1. Install dependencies
pip install -r requirements.txt

# 2. Verify the server's SSH host-key fingerprint through a trusted channel,
# then connect once with OpenSSH to record the verified key in known_hosts.
ssh mostak@100.81.129.68

# 3. Run the read-only audit. It prompts for a password if no SSH key is supplied.
python scripts/server-audit.py
```

---

## 📦 Required Python Packages

All SSH automation, remote telemetry collection, and container inspection scripts require the following packages specified in `requirements.txt`:

| Package | Purpose | Why It's Needed |
| :--- | :--- | :--- |
| **`paramiko`** | SSHv2 protocol client | Allows headless, programmatic command execution and SFTP file transfer over SSH. |
| **`cryptography`** | Modern cryptographic algorithms | Handles ED25519/RSA host keys and cipher negotiation with OpenSSH. |
| **`bcrypt`** | Password & key derivation | Used by Paramiko for encrypted SSH key handling. |

Installation command:
```bash
pip install -r requirements.txt
```

---

## 🛡️ Agent Safety & Rules of Engagement

Every AI agent operating on this server **MUST** adhere to the following safety constraints:

### 1. Default to Non-Destructive (Read-Only) Inspection
- Gather diagnostics first using safe commands: `uptime`, `docker ps`, `free -h`, `df -h`, `ss -tulpn`, `journalctl -n 50`.
- **NEVER** run unvetted disk formatting commands (`mkfs`, `fdisk`, `dd`).
- **NEVER** perform blanket directory deletions (`rm -rf /` or deleting `/data/coolify`).

### 2. SSH Credentials and Privileged Commands
- Use the audit script's password prompt or a protected SSH key. Verify the server host key before first use; the script rejects unknown host keys.
- Never place a password in an SSH command, shell history, process arguments, output, or documentation. Removing a password from returned text does not undo exposure in a command.
- The audit runs read-only commands without `sudo`. Do not add a generic remote `sudo` helper. Review and authorize each privileged operation under [AGENTS.md](../AGENTS.md).
- The administrative user has Docker access, which grants root-level power. These agent instructions do not technically restrict that account; use a separate account without sudo or Docker socket access if an agent needs enforced read-only access.

### 3. Docker & Coolify Precautions
- **Do not modify `/data/coolify/source/.env`** without creating a timestamped backup copy first (`cp .env .env.bak.$(date +%s)`).
- **Do not restart `coolify-proxy` (Traefik)** unless you have verified syntax in `/data/coolify/proxy/`. Traefik is the ingress lifeline for all hosted applications.
- **Do not restart or prune volumes** without confirming that database volumes (`coolify-db`, `coolify-redis`, `mcp-memory-data`) are preserved.

### 4. Mandatory Runbook & Changelog Updates
Whenever an agent executes an emergency fix or alters system behavior:
1. Document the incident in `runbooks/inc-XXX-<slug>.md` using `runbooks/TEMPLATE.md`.
2. Add a summary entry under `CHANGELOG.md`.

---

## 🐍 Reusable Read-Only SSH Audit

Use [`scripts/server-audit.py`](../scripts/server-audit.py) instead of copying an ad hoc SSH or `sudo` snippet. It uses the local `known_hosts` file and rejects unverified host keys. The generated report can contain network and service details; review it before sharing.

---

## 🔑 Known Service Coordinates & Ports

| Service | Address / Port | Notes |
| :--- | :--- | :--- |
| **SSH** | `100.81.129.68:22` | Direct terminal access |
| **Coolify Dashboard** | `http://100.81.129.68:8000` | Management UI (credentials in `/data/coolify/source/.env`) |
| **MCP Memory Service** | `http://100.81.129.68:8001` | Direct API endpoint |
| **Traefik Reverse Proxy** | `Ports: 80, 443, 8080` | Automatic SSL and dynamic routing |
