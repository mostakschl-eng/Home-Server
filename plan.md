# 📋 Implementation Plan: Nextcloud Media Compression Engine

---

## 🎯 Objective
Build a lightweight, automated background media optimization service on `homeserver` that automatically compresses uploaded images and videos in Nextcloud, strips metadata (EXIF/GPS), and updates Nextcloud's PostgreSQL database (`oc_filecache`) without impacting server performance or user experience.

---

## 🏗️ Architecture & Component Overview

```
[User Device (Phone/PC)]
          │ (Uploads original photo/video at full speed)
          ▼
[Nextcloud Storage on Disk]
(/var/lib/docker/volumes/syo7laqurhdbmmidrumfckbb_nextcloud-data/_data/<user>/files/...)
          │
          ▼ (File safely saved to disk)
[Media Optimizer Engine (Python)]
   ├── 🖼️ Image Pipeline: Pillow (MozJPEG / WebP / EXIF clean)
   └── 🎥 Video Pipeline: FFmpeg (H.265 / H.264 CRF 22-24, audio copy, metadata strip)
          │
          ▼ (Atomic file swap: replace original only after verification)
[Updated File on Disk]
          │
          ▼
[Nextcloud Database Sync]
(`docker exec nextcloud occ files:scan --path="/<user>/files/..."`)
```

---

## 🛡️ Key Requirements & Safety Protections

### 1. Zero-Disruption Upload Flow
* The script **only runs AFTER** a file is fully uploaded and closed on disk. It never intercepts or slows down active uploads.

### 2. Laptop CPU & Thermal Protection
* **Batch Processing**: Limited to **10 files per batch**.
* **Cool-down Interval**: 30-second pause between batches to allow CPU temperatures to normalize.
* **Single Core Limit**: Video and image encoding restricted to **1 CPU thread** (`threads=1`), leaving the remaining cores 100% free for Coolify, Traefik, Nextcloud, and MCP.
* **Idle Process Priority**: Scheduled under Linux idle scheduling classes:
  ```bash
  nice -n 19 ionice -c 3 python3 media_optimizer.py
  ```

### 3. State Tracking (No Duplicate Processing)
* A lightweight SQLite database (`/data/scripts/media_optimizer/tracker.db`) stores processed file records (`fileid`, `original_size`, `compressed_size`, `timestamp`).
* The engine queries Nextcloud's PostgreSQL database to identify new uncompressed files.

### 4. Database Integrity & Nextcloud Sync
* Once a file is compressed, the script triggers:
  ```bash
  docker exec nextcloud occ files:scan --path="/<user>/files/<relative_path>"
  ```
* Nextcloud updates `oc_filecache` table (`size`, `etag`, `mtime`) immediately, reflecting the saved storage space in the web UI and mobile apps.

---

## 📐 Quality Standards & Target Compression

| Media Type | Formats | Tool / Codec | Quality Setting | Target Savings | Visual Impact |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Images** | JPG, JPEG, PNG, WEBP | `Pillow` (MozJPEG) | Quality: 82–85, optimize=True | **60% – 75%** | Visually lossless |
| **Videos** | MP4, MOV, MKV | `FFmpeg` (libx265 / libx264) | CRF: 22–24, audio copy (`-c:a copy`) | **50% – 70%** | Same FPS, same resolution |

---

## 🚀 Step-by-Step Rollout Stages (To Execute Later)

1. **Stage 1: Standalone Test Bench (`test_optimizer.py`)**
   * Run manually on 1 sample photo and 1 sample video.
   * Generate a before/after scoreboard (size, percentage saved, processing time).
2. **Stage 2: User Quality Validation**
   * User verifies the visual output on phone/browser to approve the compression quality.
3. **Stage 3: Integration with Nextcloud Storage**
   * Connect tracking database to Nextcloud's user files directory.
   * Implement automated `occ files:scan` hook.
4. **Stage 4: Background Service (`systemd` / `cron`)**
   * Deploy systemd unit with `nice -n 19 ionice -c 3` for automated, non-intrusive background runs.
