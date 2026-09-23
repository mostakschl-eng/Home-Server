# 🌐 Network Topology & Port Matrix

This document provides a breakdown of network interfaces, IP address allocations, Tailscale WireGuard mesh topology, routing tables, and the open port matrix on `homeserver`.

---

## 🖧 Network Interfaces

```text
Interface Name    State    MAC Address        IP Assignment                      Role
──────────────────────────────────────────────────────────────────────────────────────────────────
wlp2s0            UP       b8:8a:60:51:2d:77  192.168.0.149/24                   Primary Wi-Fi Uplink (Intel 8260)
enp0s31f6         DOWN     ec:8e:b5:a3:8d:1b  Unassigned (No Carrier)            Gigabit Ethernet (Intel I219-LM)
tailscale0        UP       -                  100.81.129.68/32, fd7a:115c:...    Zero-Trust WireGuard Overlay Tunnel
br-9c58423ac515   UP       a2:3c:e1:43:59:e5  10.0.1.1/24, fd92:5548:7b6::1/64   Docker Coolify Bridge Network
docker0           DOWN     7e:ff:e8:a6:9f:b3  10.0.0.1/24                        Default Docker Bridge (Dormant)
lo                UP       00:00:00:00:00:00  127.0.0.1/8, ::1/128              Loopback
```

---

## 🔒 Tailscale Zero-Trust Mesh Network

The server is integrated into an encrypted, point-to-point WireGuard mesh network managed by **Tailscale**:

```text
Account Domain:      mostak.a.nayem@
Server Node IP:      100.81.129.68 (IPv4) / fd7a:115c:a1e0::1e01:81d1 (IPv6)
MagicDNS Name:       homeserver
Transport Port:      41641 / UDP
```

### Peer Device Inventory

| Node Name | Tailscale IPv4 | OS Platform | Status | Role |
| :--- | :--- | :--- | :--- | :--- |
| **`homeserver`** | `100.81.129.68` | Linux (Ubuntu) | **Self (Active)** | Central compute / PaaS host |
| **`desktop-vpvd0md`** | `100.104.22.76` | Windows | Offline (Last seen 10h) | Primary workstation |
| **`galaxy-s22-ultra`** | `100.112.83.33` | Android | Offline (Last seen 1h) | Mobile management client |
| **`user`** | `100.104.142.55` | Windows | **Active (Connected)** | Active management workstation |

### WireGuard Offload Optimization (`tailscale-gro`)
Tailscale packet forwarding is optimized via a custom systemd service running `/usr/sbin/ethtool -K wlp2s0 rx-udp-gro-forwarding on rx-gro-list off`. This offloads UDP GRO (Generic Receive Offload) processing to kernel hardware pipelines, preventing CPU throttling during high-bandwidth transfers.

---

## 🎯 Listening Port Matrix

| Port | Protocol | Binding Address | Process / Service | Purpose |
| :--- | :--- | :--- | :--- | :--- |
| **22** | TCP | `0.0.0.0`, `[::]` | `sshd` (PID 3136) | Secure Shell administrative access |
| **80** | TCP | `0.0.0.0`, `[::]` | `docker-proxy` -> `coolify-proxy` | HTTP ingress / automatic HTTPS redirect |
| **443** | TCP/UDP | `0.0.0.0`, `[::]` | `docker-proxy` -> `coolify-proxy` | HTTPS ingress / HTTP/3 QUIC |
| **6001** | TCP | `0.0.0.0`, `[::]` | `docker-proxy` -> `coolify-realtime` | Soketi WebSocket frontend |
| **6002** | TCP | `0.0.0.0`, `[::]` | `docker-proxy` -> `coolify-realtime` | Soketi WebSocket administrative channel |
| **8000** | TCP | `0.0.0.0`, `[::]` | `docker-proxy` -> `coolify` | Coolify management web UI |
| **8001** | TCP | `0.0.0.0`, `[::]` | `docker-proxy` -> `mcp-memory` | MCP Memory Service REST/API endpoint |
| **8080** | TCP | `0.0.0.0`, `[::]` | `docker-proxy` -> `coolify-proxy` | Traefik diagnostic/API port |
| **41641** | UDP | `0.0.0.0`, `[::]` | `tailscaled` (PID 1213) | Tailscale direct WireGuard peer communication |
| **45334** | TCP | `100.81.129.68` | `tailscaled` (PID 1213) | Tailscale node control socket |
| **53** | TCP/UDP | `127.0.0.53`, `127.0.0.54` | `systemd-resolved` (PID 630) | Local stub DNS resolver |
| **323** | UDP | `127.0.0.1`, `[::1]` | `chronyd` (PID 1270) | Local NTP chrony tracking socket |

---

## 🚦 IP Routing Table (`ip r`)

```text
default via 192.168.0.1 dev wlp2s0 proto dhcp src 192.168.0.149 metric 600
10.0.0.0/24 dev docker0 proto kernel scope link src 10.0.0.1 linkdown
10.0.1.0/24 dev br-9c58423ac515 proto kernel scope link src 10.0.1.1
192.168.0.0/24 dev wlp2s0 proto kernel scope link src 192.168.0.149 metric 600
192.168.0.1 dev wlp2s0 proto dhcp scope link src 192.168.0.149 metric 600
```
