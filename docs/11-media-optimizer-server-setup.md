# Nextcloud Media Optimizer: Future Server Setup

**Status: instructions only.** Nothing in this guide has been run on the server. It deliberately does not contain a password, a live data-volume path, a guessed container name, or a configured endpoint. Do not run the installation steps until the local behavior is accepted and server installation is requested.

## What will run

The server will run one compiled Linux amd64 Go binary on the Nextcloud host at 5:00 AM daily. Cron starts the binary; Go handles the queue and workflow. External programs handle media formats: MozJPEG `jpegtran` for JPEG, `oxipng` for PNG, `webpmux` for WebP metadata chunks, and FFmpeg/ffprobe for video stream-copy metadata removal. The binary also needs the Docker CLI to run Nextcloud `occ files:scan` inside the existing container. Go itself is needed to build the binary, not to run it.

The process is sequential (one file at a time) and stores its SQLite queue/history under `/var/lib/media-optimizer/`. It does not create a Docker container, set Docker CPU/memory limits, or alter existing containers. It walks the WebDAV tree for eligible untagged media on a clean queue; last-run timestamps are recorded but are not currently used as an incremental 24-hour filter.

## Values that must be discovered on the server

Before installation, establish these from the actual server configuration; do not infer them from old documentation:

- Nextcloud base URL and a dedicated Nextcloud app password for the service account.
- The exact host path mounted as `/data` in the live Nextcloud container.
- The live Nextcloud container name, `occ` path, and container service user (currently expected as `abc`, but verify).
- Whether `opt` and `skip` are existing, unique, assignable Nextcloud system tags for this account.
- Installed versions/paths for `jpegtran`, `oxipng`, `webpmux`, FFmpeg, ffprobe, and Docker.
- File ownership and permissions needed by the process to read and replace user files and restore their original owner/mode.
- Available CPU/RAM and an acceptable execution window; this runs on the host and can use CPU while processing video.

The host path and container data must refer to the same Nextcloud storage. If the actual mount differs, update configuration deliberately before running anything.

## Installation sequence (future)

1. Build for the server's verified architecture from the reviewed project source:

   ```sh
   CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o media-optimizer .
   ```

2. Install the binary at `/usr/local/sbin/media-optimizer`, owned by root and executable. Create `/var/lib/media-optimizer/` for the SQLite database with restrictive permissions for the service account.
3. Create `/etc/media-optimizer.env` with the values established above and mode `0600`. Populate it interactively on the server; never put the app password in a shell command, cron line, source file, or Git.
4. Verify the configuration without contacting Nextcloud:

   ```sh
   /usr/local/sbin/media-optimizer -check-config
   ```

5. Verify the runtime dependencies, permissions, tag visibility/assignability, and the exact host-to-container data mount. Confirm the process can invoke Docker and `occ` with the configured container user.
6. Run a deliberately small live test using a dedicated Nextcloud test account/library that contains only approved copies of sample files. Discovery scans the configured account recursively, and the program has no folder allowlist or dry-run mode; do not point this test at the full personal library. Check the source file, Nextcloud reported size/ETag, scan result, and `opt` tag afterward. `-check-config` only validates required settings and does not contact Nextcloud.
7. Only after the small live test is accepted, install a 5 AM cron entry. Use an explicit PATH for the external tools and load the protected environment file:

   ```cron
   0 5 * * * set -a; . /etc/media-optimizer.env; set +a; export PATH=/usr/local/mozjpeg/bin:/usr/local/bin:/usr/bin:/bin; /usr/local/sbin/media-optimizer run >> /var/log/media-optimizer.log 2>&1
   ```

   This example assumes the env file contains shell-compatible assignments. Confirm the actual binary paths and cron shell before installing it. The program's SQLite `flock` prevents concurrent optimizer runs; it does not prevent Nextcloud uploads or acquire Nextcloud's transactional file lock.
8. Inspect the first scheduled run's log, SQLite run status/timestamps, Nextcloud file metadata, and assigned tags. Remove or disable the cron entry if checks show stale metadata or unexpected file changes.

## Operational constraints

- The optimizer must run as a host process with access to the Nextcloud host data directory and Docker socket. Docker access grants broad host power under the repository's agent rules; prefer a dedicated service identity and narrowly controlled deployment.
- Keep the SQLite database on a local filesystem. The process writes one file at a time and uses same-directory atomic replacement, but direct disk access still bypasses Nextcloud transactional locking.
- A scheduled run may discover and process a large backlog on its first clean run. Sequential processing protects against parallel load, but it does not impose a maximum runtime or host CPU cap.
- This procedure covers only the optimizer. It does not install or change container limits, alter other services, or change Nextcloud/PostgreSQL schema.
