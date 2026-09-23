# 📋 [INC-003] Docker Disk Growth & Automated Maintenance Pipeline

---

## 📌 Metadata
- **Incident ID**: `INC-003`
- **Date / Timestamp**: `2026-09-22 17:00 UTC`
- **Impacted Service(s)**: Docker Engine (`dockerd`), Root Partition (`/`)
- **Severity Level**: Medium (Proactive prevention against disk exhaustion)
- **Lead Engineer**: `mostak`
- **Resolution Status**: **Resolved**

---

## 1. ⚠️ Problem Statement & Symptoms
With Coolify frequently building, pulling, and updating Docker images, unused intermediate build layers, stopped test containers, and untagged `<none>` images accumulated in `/var/lib/docker/overlayfs/`. Left unchecked, Docker image sprawl is the #1 cause of unprompted root volume disk exhaustion in home server environments.

---

## 2. 🔍 Root Cause Analysis (RCA)
- Docker maintains cached layers from prior container builds indefinitely unless explicitly pruned.
- Unattended package updates also leave `.deb` archives in `/var/cache/apt/archives/`.

---

## 3. 🛠️ Step-by-Step Resolution Procedures
Implemented an automated, safe weekly cleanup pipeline in `/etc/cron.weekly/docker-cleanup`.

### Pipeline Code (`/etc/cron.weekly/docker-cleanup`):
```bash
#!/bin/sh
# 1. Prune dangling build cache older than 48h
docker builder prune -a --filter "until=48h" -f >/dev/null 2>&1

# 2. Prune dangling (untagged) images
docker image prune -f >/dev/null 2>&1

# 3. For each repository, keep only the 3 newest images and remove older ones
for repo in $(docker images --format "{{.Repository}}" | sort -u | grep -v "<none>"); do
  docker images "$repo" -q | awk 'NR>3' | xargs -r docker rmi >/dev/null 2>&1
done

# 4. Remove stopped containers older than 7 days
docker container prune -f --filter "until=168h" >/dev/null 2>&1

# 5. Clean apt cache
apt-get clean >/dev/null 2>&1
```

### File Permissions:
```bash
sudo chmod +x /etc/cron.weekly/docker-cleanup
```

---

## 4. ✅ Verification & Quality Assurance
Executed manual dry-run test:

```bash
sudo /etc/cron.weekly/docker-cleanup
sudo docker system df
```

Disk space utilization remains constrained under **6%** (12 GB used out of 231 GB available).

---

## 5. 🛡️ Preventative Measures
- Cron execution occurs automatically every Sunday during standard system low-load windows.
- Retaining 3 newest images ensures that rollbacks within Coolify can occur without requiring full external image re-downloads.
