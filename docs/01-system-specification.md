# 💻 System & Hardware Specification

This document contains low-level hardware specifications, processor details, memory maps, power architecture, and operating system attributes for the `homeserver` node.

---

## 🏷️ Chassis & Machine Identity

| Property | Value | Detail |
| :--- | :--- | :--- |
| **Static Hostname** | `homeserver` | Defined in `/etc/hostname` |
| **Hardware Vendor** | HP (Hewlett-Packard) | Enterprise EliteBook line |
| **Product Model** | HP EliteBook 840 G3 | Commercial business laptop platform |
| **Chassis Form Factor** | Laptop 💻 | Includes integrated battery (acts as hardware UPS) |
| **Hardware SKU** | `W8F62UP#ABA` | |
| **Machine ID** | `7a7aae592e0d4ddba2f5098c8f9eac25` | Stored in `/etc/machine-id` |
| **System BIOS / Firmware** | HP N75 Ver. 01.52 (2021-04-20) | UEFI 64-bit mode |

---

## ⚙️ Processor (CPU) Architecture

```text
Processor Model:      Intel(R) Core(TM) i5-6300U CPU @ 2.40GHz
Microarchitecture:    Skylake (6th Generation Intel Core)
Cores / Threads:      2 Physical Cores / 4 Threads (Hyper-Threading enabled)
Base Clock:           2.40 GHz
Max Turbo Frequency:  3.00 GHz
Min Frequency:        400 MHz (Power-saving idle state)
Instruction Set:      x86_64 (64-bit mode, little endian)
BogoMIPS:             4999.90
Virtualization:       Intel VT-x supported
```

### Cache Hierarchy
- **L1 Data Cache (L1d)**: 64 KiB (2 instances × 32 KiB)
- **L1 Instruction Cache (L1i)**: 64 KiB (2 instances × 32 KiB)
- **L2 Unified Cache**: 512 KiB (2 instances × 256 KiB)
- **L3 Shared Cache**: 3.0 MiB (Smart Cache)

### Hardware Vulnerability Mitigations
All modern CPU vulnerability mitigations are active in kernel space:
- **Meltdown**: Mitigation: Kernel Page Table Isolation (`PTI`)
- **Spectre v1**: Mitigation: Usercopy/swapgs barriers & `__user` pointer sanitization
- **Spectre v2**: Mitigation: Indirect Branch Restricted Speculation (`IBRS`)
- **MDS / MMIO**: Mitigation: Clear CPU buffers (`SMT vulnerable`)
- **SRBDS**: Mitigation: Microcode mitigation enabled

---

## 🧠 Memory (RAM) Profile

```text
Total System RAM:     16 GB (15,725,636 kB reported by /proc/meminfo)
Usable Physical:      14.9 GiB
Active Footprint:     ~1.4 GiB (Standard idle with Coolify + Traefik + Postgres + Redis + MCP)
Available / Buffer:   ~13.5 GiB
Swap Partition:       4.0 GiB (/swap.img, swappiness default 60)
Swap Utilization:     0 Bytes (Healthy, no memory pressure)
```

---

## 🐧 Operating System & Kernel

| Parameter | Configuration |
| :--- | :--- |
| **Distribution** | Ubuntu 26.04.1 LTS |
| **Release Codename** | *Resolute Raccoon* |
| **Kernel Release** | `7.0.0-31-generic` |
| **Kernel Architecture** | `x86_64 GNU/Linux` |
| **Systemd Version** | Modern systemd with networkd + resolved |
| **Default Shell** | GNU Bash 5.x |
| **System Timezone** | `Etc/UTC` (UTC +0000, NTP synchronized via `chrony`) |

---

## 🔌 Thermal & Power Profile

Running a laptop as a home server provides significant benefits:
1. **Built-in Uninterruptible Power Supply (UPS)**: The internal lithium-ion battery safeguards against temporary power cuts or voltage drops without requiring an external battery unit.
2. **Thermal Daemon**: `thermald.service` is running to ensure dynamic thermal management and fan speed modulation.
3. **Power Consumption**: The Intel Core i5-6300U has a thermal design power (TDP) of only **15 Watts**, idling between 4W - 8W, ensuring 24/7 cost-effective and whisper-quiet operation.
