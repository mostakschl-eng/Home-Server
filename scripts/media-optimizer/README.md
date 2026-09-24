# Nextcloud media optimizer

This is a single Go binary intended to run beside Nextcloud on the same Linux server. It has not been connected to or installed on the server. Build and use it locally with a mock/test environment first; deployment values in `.env.example` are intentionally placeholders.

## What it does

At each scheduled run, it records the run start/status in SQLite, recovers unfinished queue rows, and then discovers untagged media with recursive WebDAV `PROPFIND` calls. It also records run finish time and last successful finish. Nextcloud's WebDAV search does not provide a documented negative system-tag filter, so the script enumerates directories and filters the `oc:tags` property locally. It fails closed if Nextcloud does not return tags; it never assumes that an untagged file is safe to process.

Files are handled one at a time:

1. `.dng`, camera RAW variants, `.psd`, and `.psb` receive the existing `skip` system tag and are not opened by a media tool.
2. JPEG uses MozJPEG `jpegtran` losslessly. It keeps JFIF, ICC, Adobe color transform, and EXIF orientation, while removing GPS and other EXIF, XMP, IPTC, and comment data.
3. PNG uses `oxipng -o 4 --strip safe`; pixel data stays lossless.
4. WebP uses `webpmux` to strip EXIF and XMP chunks without decoding or re-encoding; animation and ICC color profile remain intact.
5. Video uses FFmpeg stream copy (`-c copy`) and drops container/stream metadata and chapters. It does not re-encode audio or video, so size savings may be small.
6. The output is decoded for validation and image dimensions are checked. A replacement is made only when the output is smaller. A same-size or larger valid result leaves the original in place and is still marked `opt` after the scan.
7. Before replacement, the script checks the Nextcloud ETag and local size/mtime again. It writes a temporary file beside the original, restores its mode and owner, then atomically renames it into place.
8. It runs `docker exec ... occ files:scan --path user/files/path`, assigns the `opt` system tag, then deletes the completed SQLite row.

Intermediate rows use `pending → processing → validated → replaced → scanned → tagged`. At startup, `tagged` rows are removed and any interrupted intermediate row is reset to `pending`, per the agreed recovery flow. JPEG/PNG/WebP processing is lossless and video streams are copied, so repeating after a crash is safe for decoded image pixels or encoded video/audio streams. SQLite stores `last_run_started_at`, `last_successful_run_at`, and run status; it does not store file contents.

## Important limits

- Direct writes to Nextcloud's data directory bypass Nextcloud's transactional file lock. The script skips files Nextcloud reports locked and rechecks the ETag/mtime before replacement, but it does not acquire Nextcloud's transaction lock and cannot eliminate the race between that final check and atomic rename. The 5 AM schedule reduces contention; it is not a lock guarantee. Nextcloud documents its locking as higher-level than filesystem locking ([locking details](https://docs.nextcloud.com/server/stable/admin_manual/configuration_files/files_locking_transactional.html)).
- The host data-volume path, live container name, Nextcloud URL, service user, and app password need to be set on the server. Project docs only identify the in-container `/data` mount, not its host path. Do not guess the host path.
- Ensure the existing system tags `opt` and `skip` are visible and assignable by this Nextcloud account. The script checks that both tags exist before processing; assignment permission is confirmed by Nextcloud when a file is tagged.
- Required server tools: Go is needed only to build; the server runs the standalone binary. Runtime tools are MozJPEG `jpegtran`, `oxipng`, `webpmux`, FFmpeg/ffprobe, Docker CLI, `chown`, and access to the Nextcloud container. All paths can be configured through environment variables.
- Run the process with permission to read/write the host data volume, set ownership back to the original file owner (`chown --reference`), and access the Docker socket. A dedicated account with narrowly granted access is preferable to broad administrative access.
- This program does not install packages, create the tags, configure a timer, change container limits, or contact the server during local development.

## Build and test

With Go 1.24 or later:

```sh
go test ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o media-optimizer .
```

Copy the binary and a root-readable environment file to the server only after you have locally reviewed and tested the behavior. Run `media-optimizer -check-config` to check that required settings are present without contacting Nextcloud. `media-optimizer run` performs a live run.

Store the environment values in `/etc/media-optimizer.env` with owner root and mode `0600`. Example daily cron entry after deployment is configured:

```cron
0 5 * * * set -a; . /etc/media-optimizer.env && set +a && /usr/local/sbin/media-optimizer run >> /var/log/media-optimizer.log 2>&1
```

The binary uses a Linux `flock` lock file, so an overlapping invocation exits instead of processing the same queue concurrently. Keep the SQLite file on a local filesystem, not a network share.

## Nextcloud API note

Nextcloud documents WebDAV `PROPFIND` for file properties, including file ID, ETag, size, and tags. Its documented WebDAV search properties do not include a “files missing both tags” predicate. This script therefore lists each directory, reads tags, and selects untagged supported media itself; it never queries or updates Nextcloud's PostgreSQL tables directly.

