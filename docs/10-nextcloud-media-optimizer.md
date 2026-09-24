# Nextcloud Media Optimizer: Workflow and Status

## Current status

The Go source and a Linux amd64 binary are in `scripts/media-optimizer/`. The program has **not** been connected to, installed on, or run against the server. It has no configured server credentials or real server paths. The former `scripts/media-optimizer-test/` folder was a separate experiment; its seven original sample files are being kept in `scripts/media-optimizer/testdata/input/`. Prior generated outputs and bundled FFmpeg/Go tools are disposable test artifacts, not runtime requirements.

The Go unit tests and a Linux build have been run locally. These checks do not prove the WebDAV, Docker/`occ`, tag, or file replacement flow against the live Nextcloud server; that integration remains untested. No server change has been made.

## Agreed daily workflow

The intended schedule is 5:00 AM on the Nextcloud host. The scheduler is not installed yet. Each run does the following:

1. Open the local SQLite queue and record run start/status. A Linux `flock` lock prevents two optimizer processes from using the queue at once.
2. Recover rows left by an interrupted run: remove completed `tagged` rows and reset unfinished stages to `pending` so the file is checked and processed again.
3. If pending work exists, resume it first. Otherwise, recursively list Nextcloud files through WebDAV, read their tags, and queue eligible files that have neither `opt` nor `skip`. The `opt`/`skip` system tags in Nextcloud are the durable per-file result.
4. Handle exactly one queued file at a time. Check its type and Nextcloud lock status, wait until size and modification time are stable, and read it from the host-mounted Nextcloud data directory.
5. Tag `.dng`, camera RAW formats, `.psd`, and `.psb` with `skip` without opening them in a media tool.
6. Process supported types without changing the format: JPEG uses lossless MozJPEG `jpegtran` while preserving ICC/color and orientation data and removing selected metadata; PNG uses lossless `oxipng`; WebP has EXIF/XMP chunks removed without re-encoding; video uses FFmpeg stream copy and removes container metadata/chapters without re-encoding streams.
7. Decode/inspect the temporary output. Replace the original only if validation succeeds and the output is smaller. Recheck the remote ETag and local file state before writing a same-directory temporary file and atomically replacing the original.
8. Run `occ files:scan` for that file so Nextcloud refreshes its file cache/database metadata, assign `opt`, then remove the completed SQLite queue row. If the output is not smaller, leave the original untouched and still scan/tag it as processed.
9. Record the run finish time and success/failure in SQLite.

SQLite stages are `pending → processing → validated → replaced → scanned → tagged`. SQLite is a resumable work queue and run-history record; Nextcloud's tags are the permanent per-file completion/exclusion record. The stored run timestamps are audit status only: discovery currently walks the full WebDAV tree when the queue is empty. It does not use a previous 5 AM timestamp to query only the last 24 hours.

## Resource and concurrency behavior

The optimizer processes one file at a time and does not run inside a new Docker container. Thus, there is no optimizer container memory/CPU limit to configure; the process uses host resources while running. `flock` prevents overlapping copies of this program, not other Nextcloud activity. A 5 AM run may reduce user activity but does not guarantee nobody is uploading a file.

Direct host-disk replacement bypasses Nextcloud's transactional file lock. The program skips files Nextcloud reports locked and repeats ETag, size, and modification-time checks immediately before replacement, but a narrow race remains between the final check and rename. Nextcloud describes transactional locking in its [file locking documentation](https://docs.nextcloud.com/server/stable/admin_manual/configuration_files/files_locking_transactional.html). Validate the live behavior and accept this limitation before deployment.

## What the Windows lock file means

`lock_windows.go` is the Windows build/test implementation of the process lock. It creates an exclusive lock file so concurrent local Windows runs cannot use the same SQLite queue. The intended Ubuntu server uses `lock_unix.go`, which uses Linux `flock`; the Windows implementation is not deployed there. Neither lock coordinates with Nextcloud's own per-file transactional locking.

## Local checks

Run from `scripts/media-optimizer/` with Go 1.24 or later:

```sh
go test ./...
go vet ./...
gofmt -d *.go
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o media-optimizer-linux-amd64 .
```

The sample files are private local inputs and are ignored by Git. The Go unit tests are not an end-to-end media run. Do not interpret earlier CRF 21/24 lossy sample outputs as results from this current lossless/stream-copy program.

## Next step

Use [the server setup procedure](11-media-optimizer-server-setup.md) only when server installation is explicitly requested. Until then, keep the source, sample originals, and Linux binary in this project; do not install a cron entry, copy credentials, or change Nextcloud.
