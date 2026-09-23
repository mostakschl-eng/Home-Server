#!/usr/bin/env python3
"""
homeserver Automated Audit & Telemetry Collector
================================================
Non-destructive read-only audit utility for Ubuntu Home Server.
Fetches system hardware, storage, network interfaces, Docker containers,
and systemd status over SSH and outputs a clean JSON report.

Usage:
  python server-audit.py [--host 100.81.129.68] [--user mostak] [--output report.json]
"""

import argparse
import getpass
import json
import os
import sys

try:
    import paramiko
except ImportError:
    print("Error: 'paramiko' is required. Install via: pip install paramiko")
    sys.exit(1)

SAFE_COMMANDS = {
    "hostnamectl": "hostnamectl",
    "uptime": "uptime",
    "os_release": "cat /etc/os-release",
    "kernel": "uname -a",
    "lscpu": "lscpu",
    "memory": "free -h",
    "disks_lsblk": "lsblk -e7 -o NAME,FSTYPE,FSVER,SIZE,MOUNTPOINT,FSUSED,FSUSE%,MODEL",
    "filesystem_df": "df -hT -x tmpfs -x devtmpfs -x squashfs",
    "network_interfaces": "ip -br a",
    "ip_routes": "ip r",
    "tailscale_status": "tailscale status 2>&1 || true",
    "listening_ports": "ss -tulpn 2>&1 || netstat -tuln 2>&1 || true",
    "systemd_running": "systemctl list-units --type=service --state=running --no-pager",
    "systemd_failed": "systemctl --failed --no-pager",
    "docker_version": "docker version 2>&1 || true",
    "installed_snaps": "snap list 2>&1 || true"
}

def main():
    parser = argparse.ArgumentParser(description="Audit Ubuntu Home Server via SSH")
    parser.add_argument("--host", default="100.81.129.68", help="Target Server IP or Hostname")
    parser.add_argument("--port", type=int, default=22, help="SSH Port (Default: 22)")
    parser.add_argument("--user", default="mostak", help="SSH Username")
    parser.add_argument("--key", default=None, help="Path to Private SSH Key")
    parser.add_argument("--output", default="server_audit_report.json", help="Output file path")
    args = parser.parse_args()

    password = os.environ.get("SERVER_SSH_PASS")
    if not password and not args.key:
        password = getpass.getpass(f"Enter SSH password for {args.user}@{args.host}: ")

    print(f"Connecting to {args.user}@{args.host}:{args.port}...")
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

    try:
        client.connect(
            hostname=args.host,
            port=args.port,
            username=args.user,
            password=password,
            key_filename=args.key,
            timeout=15
        )
        print("Connected successfully!")
    except Exception as e:
        print(f"Connection failed: {e}")
        sys.exit(1)

    results = {}
    for key, cmd in SAFE_COMMANDS.items():
        print(f"  -> Collecting: {key}")
        try:
            stdin, stdout, stderr = client.exec_command(cmd, timeout=30)
            results[key] = {
                "command": cmd,
                "stdout": stdout.read().decode("utf-8", errors="replace").strip(),
                "stderr": stderr.read().decode("utf-8", errors="replace").strip(),
                "status": stdout.channel.recv_exit_status()
            }
        except Exception as e:
            results[key] = {"command": cmd, "error": str(e)}

    client.close()

    with open(args.output, "w", encoding="utf-8") as f:
        json.dump(results, f, indent=2)

    print(f"\nAudit complete! Report written to {args.output}")

if __name__ == "__main__":
    main()
