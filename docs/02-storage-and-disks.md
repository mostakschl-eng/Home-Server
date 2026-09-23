# 💾 Storage & Filesystem Architecture

This document provides a breakdown of the physical drive, LVM partition layout, filesystem mounts, swap space, and volume mapping on `homeserver`.

---

## 💽 Physical Drive Details

The system operates from a single high-reliability SATA Solid State Drive:

```text
Drive Model:       SanDisk SD7SN6S-256G-1006
Serial Number:     162223805217
Interface:         SATA 6.0 Gb/s (AHCI mode)
Form Factor:       M.2 2280 SATA
Total Capacity:    256 GB (238.5 GiB raw binary capacity)
Device Node:       /dev/sda
Disk By-ID:        /dev/disk/by-id/ata-SanDisk_SD7SN6S-256G-1006_162223805217
```

---

## 📐 Partition Map & Logical Volume Manager (LVM)

The storage layout employs **Logical Volume Management (LVM2)** for flexible volume expansion and snapshot capability:

```text
/dev/sda (238.5G SATA SSD)
 ├─/dev/sda1 (1.0G, FAT32) ──────────> Mounted at: /boot/efi (UEFI Boot partition)
 ├─/dev/sda2 (2.0G, ext4) ───────────> Mounted at: /boot (Kernel & Initrd storage)
 └─/dev/sda3 (235.4G, LVM2 Member)
     └─ ubuntu-vg (Volume Group)
         └─ ubuntu--vg-ubuntu--lv (235.4G) ──> Mounted at: / (ext4 Root Filesystem)
```

### Mount Table (`df -hT`)

| Filesystem Device | Mount Point | Type | Size | Used | Available | Use % |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `/dev/mapper/ubuntu--vg-ubuntu--lv` | `/` (Root) | ext4 | 231 GB | 12 GB | 208 GB | **6%** |
| `/dev/sda2` | `/boot` | ext4 | 2.0 GB | 97 MB | 1.7 GB | **6%** |
| `/dev/sda1` | `/boot/efi` | vfat | 1.1 GB | 6.4 MB | 1.1 GB | **1%** |
| `efivarfs` | `/sys/firmware/efi/efivars` | efivarfs | 118 KB | 43 KB | 71 KB | 38% |

---

## 📁 System Mounts (`/etc/fstab`)

The system uses deterministic disk UUIDs and Device Mapper IDs to ensure reliable booting:

```fstab
# / was on /dev/ubuntu-vg/ubuntu-lv during installation
/dev/disk/by-id/dm-uuid-LVM-l13VnzmgEO9tAjT2kAzf6xtXZL7DbJQrD19QcTFmSdaCAYXeGa94iK1i1AcEuT9w / ext4 defaults 0 1

# /boot was on /dev/sda2 during installation
/dev/disk/by-uuid/fc3a3322-a749-4dc9-8171-afd765144320 /boot ext4 defaults 0 1

# /boot/efi was on /dev/sda1 during installation
/dev/disk/by-uuid/51C9-0398 /boot/efi vfat defaults 0 1

# 4 GB Swap File
/swap.img none swap sw 0 0
```

---

## 📦 Application Volumes & Data Directories

All mission-critical service data resides within two primary directories on `/`:

### 1. Coolify Root Directory (`/data/coolify`)
- `/data/coolify/source/`: Contains Coolify backend environment `.env` and production docker-compose specifications.
- `/data/coolify/proxy/`: Contains Traefik reverse proxy dynamic configuration and SSL certificates.
- `/data/coolify/applications/`: Houses application definitions (including Docker Compose configurations).
- `/data/coolify/ssh/`: SSH automation keys used by Coolify to deploy to local and remote nodes.
- `/data/coolify/sentinel/`: Telemetry state database for Sentinel monitoring.

### 2. Docker Volumes Directory (`/var/lib/docker/volumes/`)
- `coolify-db/_data`: PostgreSQL 15 database files (Coolify system catalog and credentials).
- `coolify-redis/_data`: Redis database dump file (`dump.rdb`).
- `00lpnw9fxnqwf6f8lzonywhk-mcp-memory-data/_data`: SQLite database holding the MCP AI Memory entries.

---

## 🛡️ Drive Health & Trim Strategy

- **SSD TRIM**: Handled automatically via systemd `fstrim.timer` weekly to ensure long-term write endurance on the SanDisk SSD.
- **Disk Space Headroom**: With only 12 GB used out of 231 GB available (~6%), the server has over 200 GB of free space for container images, database growth, and model weights.
