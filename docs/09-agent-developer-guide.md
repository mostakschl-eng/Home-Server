# 🤖 AI Agent & Developer Operating Guide

This guide establishes standard operating procedures for **AI coding agents (e.g. Antigravity, Claude, Gemini)** and software engineers interacting with the `homeserver` node. 

Follow this protocol to inspect, debug, manage, or deploy services safely without trial-and-error or risking downtime.

---

## ⚡ Quickstart for AI Agents

To work with this server, install the required automation dependencies:

```bash
# 1. Install dependencies
pip install -r requirements.txt

# 2. Configure environment credentials (recommended for automated sessions)
export SERVER_SSH_HOST="100.81.129.68"
export SERVER_SSH_PORT="22"
export SERVER_SSH_USER="mostak"
export SERVER_SSH_PASS="<server-password>"

# On Windows PowerShell:
$env:SERVER_SSH_HOST="100.81.129.68"
$env:SERVER_SSH_PORT="22"
$env:SERVER_SSH_USER="mostak"
$env:SERVER_SSH_PASS="<server-password>"

# 3. Run automated safe audit to refresh server status
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

### 2. Sudo Execution Protocol
The primary user (`mostak`) requires a password for `sudo` operations:
- For programmatic SSH execution via Python Paramiko:
  ```python
  # Safe non-interactive sudo execution template
  stdin, stdout, stderr = client.exec_command(f"echo '{password}' | sudo -S <command>")
  output = stdout.read().decode('utf-8').replace(f"[sudo] password for {user}: ", "").strip()
  ```
- **Filter Secrets**: Always sanitize and strip passwords from returned logs before outputting them to documents or chat transcripts.

### 3. Docker & Coolify Precautions
- **Do not modify `/data/coolify/source/.env`** without creating a timestamped backup copy first (`cp .env .env.bak.$(date +%s)`).
- **Do not restart `coolify-proxy` (Traefik)** unless you have verified syntax in `/data/coolify/proxy/`. Traefik is the ingress lifeline for all hosted applications.
- **Do not restart or prune volumes** without confirming that database volumes (`coolify-db`, `coolify-redis`, `mcp-memory-data`) are preserved.

### 4. Mandatory Runbook & Changelog Updates
Whenever an agent executes an emergency fix or alters system behavior:
1. Document the incident in `runbooks/inc-XXX-<slug>.md` using `runbooks/TEMPLATE.md`.
2. Add a summary entry under `CHANGELOG.md`.

---

## 🐍 Reusable Agent Python SSH Automation Snippet

AI agents can directly execute this standalone Python snippet to safely run remote commands on the server:

```python
import os
import paramiko

HOST = os.environ.get("SERVER_SSH_HOST", "100.81.129.68")
USER = os.environ.get("SERVER_SSH_USER", "mostak")
PASS = os.environ.get("SERVER_SSH_PASS")  # Provide password via env
PORT = int(os.environ.get("SERVER_SSH_PORT", "22"))

def execute_remote(cmd: str, use_sudo: bool = False) -> str:
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(hostname=HOST, port=PORT, username=USER, password=PASS, timeout=15)
    
    if use_sudo:
        full_cmd = f"echo '{PASS}' | sudo -S {cmd}"
    else:
        full_cmd = cmd
        
    stdin, stdout, stderr = client.exec_command(full_cmd, timeout=30)
    out = stdout.read().decode("utf-8", errors="replace")
    out = out.replace(f"[sudo] password for {USER}: ", "").strip()
    client.close()
    return out

# Example: Check running containers
if __name__ == "__main__":
    print(execute_remote("docker ps --format 'table {{.Names}}\t{{.Status}}'", use_sudo=True))
```

---

## 🔑 Known Service Coordinates & Ports

| Service | Address / Port | Notes |
| :--- | :--- | :--- |
| **SSH** | `100.81.129.68:22` | Direct terminal access |
| **Coolify Dashboard** | `http://100.81.129.68:8000` | Management UI (credentials in `/data/coolify/source/.env`) |
| **MCP Memory Service** | `http://100.81.129.68:8001` | Direct API endpoint |
| **Traefik Reverse Proxy** | `Ports: 80, 443, 8080` | Automatic SSL and dynamic routing |
