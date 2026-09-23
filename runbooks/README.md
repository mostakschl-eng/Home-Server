# 🛠️ Operations & Incident Runbooks

This directory contains **Standard Operating Procedures (SOPs)** and **Incident Post-Mortems** for `homeserver`.

Following SRE best practices modeled after hyperscaler reliability engineering (Google / Microsoft SRE), every operational anomaly, emergency intervention, and architectural fix is recorded with root-cause analysis, exact reproduction commands, and preventative measures.

---

## 📑 Runbook & Incident Directory

| ID | Title | Component | Severity | Resolution Summary |
| :--- | :--- | :--- | :--- | :--- |
| **[INC-001](inc-001-mcp-onnx-model-s3-timeout.md)** | MCP Memory ONNX Model S3 Download Timeout | `mcp-memory-service` | High | Pre-downloaded model on host and injected into container cache |
| **[INC-002](inc-002-tailscale-udp-gro-optimization.md)** | Tailscale WireGuard Throughput Bottleneck | `tailscaled`, `wlp2s0` | Medium | Created custom systemd service enabling UDP GRO forwarding |
| **[INC-003](inc-003-docker-disk-exhaustion-prevention.md)** | Docker OverlayFS Build Cache & Layer Bloat | `dockerd`, `cron` | Medium | Deployed automated weekly pruning pipeline in `/etc/cron.weekly` |
| **[INC-004](inc-004-safe-firewall-and-port-isolation.md)** | Safe Firewall Hardening & Dual-Network Access Strategy | `ufw`, `iptables`, `tailscale` | Medium | Implemented zero-lockout SOP allowing SSH on LAN & Tailscale |

---

## 📝 Documenting a New Incident or SOP

Whenever an issue occurs and is resolved:
1. Make a copy of [TEMPLATE.md](TEMPLATE.md).
2. Name the file using the format: `inc-XXX-<short-slug>.md` (e.g., `inc-004-postgres-lockup.md`).
3. Fill in all sections thoroughly:
   - **Problem Summary** & error messages.
   - **Root Cause Analysis (RCA)**.
   - **Exact Step-by-Step Resolution Commands**.
   - **Verification & Preventative Actions**.
4. Update the index table above and log an entry in [CHANGELOG.md](../CHANGELOG.md).
