# INC-008: Nextcloud upload CPU tuning and container review

## Metadata

- Date: 2026-09-26, approximately 06:40-06:46 UTC.
- Services changed: Nextcloud application settings and the `mostak` downloader scan cron.
- Authorization: user explicitly approved the discussed application changes and requested a further container review.
- Host verified: i5-6300U, two physical cores/four logical CPUs, 16 GB installed RAM, kernel `7.0.0-34-generic`, Docker `29.8.1`.
- Nextcloud: `35.0.0`, Memories `9.0.1`, Preview Generator `5.15.0`.

## Symptom and evidence

The user reported roughly 60% CPU and laptop heat during image uploads. Access logs for the two busiest hours on September 25 recorded 495 upload requests and 534 preview requests, predominantly from the iOS client. These are request counts, not unique-file or newly-generated-preview counts. Installed Memories code indexes supported files synchronously after write events. Preview Generator queues writes and runs a time-sensitive job at five-minute intervals with a default 300-second budget.

This establishes overlapping upload, indexing, and preview work; no CPU trace from the reported spike exists, so exact CPU attribution and improvement remain unmeasured. Idle samples had ample available RAM and no swap use. Preview generation was uncapped at the container level and used default maximum dimensions of 4096 pixels per axis. File locking used APCu even though Redis was running.

## Applied changes

| Setting | Before | After |
| --- | --- | --- |
| `preview_concurrency_new` | Hardware-derived default | Integer `1` |
| `preview_max_x`, `preview_max_y` | Default `4096` | Integer `2048` each |
| App `preview`: `jpeg_quality` | Default `80` | Explicit `80` |
| App `previewgenerator`: `job_max_execution_time` | Default `300` | Integer `60` |
| `memcache.locking` | `\OC\Memcache\APCu` | `\OC\Memcache\Redis` |
| Redis connection | Not configured | `redis-syo7laqurhdbmmidrumfckbb:6379` |
| Downloader scan | Every minute, obsolete `Downloads` | Every five minutes, `Media Downloads`, guarded by `flock` |

Local caching remains APCu. Core Nextcloud cron remains every five minutes. The 60-second preview budget is not a hard process kill and does not govern on-demand requests. Originals were not re-encoded. Existing cached previews were not purged, so old larger previews can remain until normal invalidation/expiry. The new limit controls creation of uncached previews, not all image-related CPU work.

Nextcloud was briefly placed in maintenance mode for the locking changes; no recent upload requests were observed beforehand. The LinuxServer `occ` wrapper stripped backslashes from the initial Redis class argument. Verification caught this, and the value was corrected with direct PHP invocation as `abc`:

```sh
docker exec -u abc nextcloud-syo7laqurhdbmmidrumfckbb php /app/www/public/occ config:system:set memcache.locking --value='\OC\Memcache\Redis'
```

Use direct PHP invocation for arguments containing backslashes or complex quoting.

## Backups and rollback

Verified nonempty/readable backups before mutation:

- Container config: `/config/tuning-backups/20260926T064015Z/config.php`, byte-compared against the original.
- Host cron: `/home/mostak/nextcloud-tuning-backups/20260926T064015Z/crontab.before`.
- Original app configuration: `preview.before.json` and `previewgenerator.before.json` in the same host backup directory.

The two app settings changed here were previously unset. To roll back this change, after accounting for any subsequent edits and active uploads:

```sh
nc=nextcloud-syo7laqurhdbmmidrumfckbb
docker exec -u abc "$nc" php /app/www/public/occ maintenance:mode --on
docker exec "$nc" cp -p /config/tuning-backups/20260926T064015Z/config.php /config/www/nextcloud/config/config.php
docker exec -u abc "$nc" php /app/www/public/occ config:app:delete previewgenerator job_max_execution_time
docker exec -u abc "$nc" php /app/www/public/occ config:app:delete preview jpeg_quality
crontab /home/mostak/nextcloud-tuning-backups/20260926T064015Z/crontab.before
docker exec -u abc "$nc" php /app/www/public/occ maintenance:mode --off
curl -fsS --max-time 10 http://nextcloud.100.81.129.68.sslip.io/status.php
```

Do not restore the whole config over unrelated later changes without reviewing them. Backups contain private configuration and must not be shared.

## Verification

- PHP config syntax passed.
- Nextcloud's bootstrapped configuration matched all changed values with the expected types.
- The resolved locking provider was `OC\Lock\MemcacheLockingProvider`, and its cache was `OC\Memcache\Redis`.
- Redis ping succeeded from inside Nextcloud.
- Status endpoint returned installed=true, maintenance=false, needsDbUpgrade=false.
- Manual `Media Downloads` scan: one folder, zero files, zero errors. This did not test a real Aria2 completion.
- Preview queue was empty. No fresh upload/preview workload was generated, so preview latency and before/after upload CPU remain unverified.
- No new Nextcloud log entries were observed during the final log check.

## Read-only Coolify and container review

All containers were healthy, with zero restart counts and zero failing health-check streaks. Four short idle CPU samples gave median Docker CPU percentages of approximately 0.96 for Coolify, 0.32 for Sentinel, 0.30 for MCP Memory, and 0.01 for Nextcloud. Docker percentages are relative to one logical CPU; they are not total-host percentages. These short idle samples do not characterize upload, embedding, or build peaks.

Live Coolify server settings: concurrent builds=2, metrics enabled, metrics sampling=10 seconds, metrics history=7 days, Sentinel push interval=60 seconds. Main Coolify PHP configuration reports 256 MB memory limit and up to 20 FPM children, but the limit is not proof that 20 processes are active. Aria2 already has 0.5 CPU and 256 MiB caps. Other running containers have no configured Docker CPU/memory caps.

Recommended next changes, not applied:

1. Reduce Coolify concurrent builds from two to one to limit overlapping build pressure on the dual-core host. This does not cap the CPU used by a single build. Prefer prebuilt images for heavy builds.
2. Consider health-check intervals around 30 seconds rather than the current 4-5 seconds for several core services. This reduces repeated probe overhead but delays failure detection; it is not a demonstrated fix for the upload spike. Persist this through the appropriate deployment configuration, not transient `docker update` overrides.
3. Configure persistent resource budgets for application workloads using measured peak usage; avoid tight database caps. Resource caps protect neighboring services but can increase latency or trigger OOM if set too low.
4. Sentinel sampling could move from 10 to 30 seconds if fine-grained history is unnecessary. Keep its heartbeat/status reporting; current measured overhead is small.

No Coolify core, proxy, database volume, firewall, or host access configuration was changed. Changes to core Coolify files require the repository's Tier 2 backup, validation, and `CONFIRM EXECUTE` procedure.

## Follow-up: safe changes and the user's 37-image upload test

At approximately 07:09 UTC on September 26, the user authorized safe additional tuning and requested investigation of slower uploads. Backups were created and validated under `/home/mostak/nextcloud-tuning-backups/20260926T070809Z/`. These contain `coolify-settings.before.json` and `previewgenerator.before.json`. A byte-verified copy of the unchanged nginx main configuration is at `/config/tuning-backups/20260926T070809Z/nginx.conf` inside Nextcloud.

Applied and verified:

- Coolify server 0: `concurrent_builds` changed from 2 to 1 through the installed Laravel `ServerSetting` model's normal save path, matching the UI persistence path. No core source file was edited, and no running build was cancelled.
- Preview Generator: `squareSizes="64 256"`, `fillWidthHeightSizes="256 1024"`, `coverWidthHeightSizes="256"`, `widthSizes=""`, `heightSizes=""`. The installed `SizeHelper` resolved five specifications: two cropped squares (64 and 256), a 256 cover preview, and 256/1024 fill previews. The previous default configuration resolved 12 specifications at the 2048-pixel maximum. This is a specification-count reduction, not a measured CPU-percentage reduction. On-demand sizes remain available, and cached previews were not purged.
- Added `/config/nginx/site-confs/upload-timing.conf` with conditional PUT/MOVE/HEAD logging to `/config/log/nginx/upload-timing.log`. It records timestamp, method, status, request byte count, request duration, and upstream duration. It does not log URL paths, client addresses, authorization headers, or file contents. Existing `/etc/logrotate.d/nginx` covers the log (weekly, 14 rotations).
- `nginx -t` passed, a graceful reload succeeded, a HEAD probe produced a valid timing record with HTTP 200, and the Nextcloud status endpoint remained healthy with maintenance off.

The installed nginx configuration has `fastcgi_request_buffering off`. Consequently, upstream duration can include receipt/streaming of the upload body and PHP worker waits; subtracting it from request duration does not isolate image-indexing CPU time. These timing fields improve observation of future requests but cannot reconstruct the previous test or prove exact network-versus-indexing attribution alone.

Observed test data:

- Exactly 37 JPG PUT requests, all HTTP 201, from the Android client; 74 HEAD checks accompanied the session (37 HTTP 200, 37 HTTP 404).
- Completion timestamps ran from 06:53:50 to 06:59:02 UTC: a 312-second first-to-last completion span, median completion gap 8 seconds, maximum gap 18 seconds. Completion gaps are not per-request execution times.
- Read-only file-cache queries found all 37 uploaded files: approximately 100 MiB total, median 2.306 MiB, maximum 6.771 MiB.
- The user confirmed this test used mobile data or another Wi-Fi, not the server's home Wi-Fi. The earlier large session predominantly used iOS and a different request pattern, so it is not a controlled before/after comparison.
- At diagnosis time the online Android peer's three Tailscale pings used Singapore DERP, with measured times of 570, 464, and 168 ms. This establishes the current route, not necessarily the route of every historical upload.
- Host `tailscale netcheck`: UDP true, IPv4 available, IPv6 unavailable, mapping does not vary by destination, UPnP/NAT-PMP observed. No blanket host UDP-connectivity failure was found.

The remote relayed route, additional Android checks, and low overall batch transfer rate are evidence-backed contributors/candidates for slowdown. Memories still extracts metadata synchronously on writes, but no historical request timings exist to quantify its share. There is no Nextcloud Docker CPU cap and no explicit change to upload concurrency. Preview semaphore limiting applies to new preview generation, not directly to each PUT. Do not assert that the preview limit caused the slower transfer.

Skipped further changes: CPU/memory caps while upload latency remains unresolved; health-check changes requiring core/proxy deployment edits for a small unmeasured gain; Sentinel sampling changes because its measured overhead is small. Sentinel remains at 10-second metric sampling and 60-second push interval. Firewall, NAT, proxy routing, and TLS were not changed.

Follow-up rollback, after reviewing any subsequent changes:

1. Restore Coolify `concurrent_builds=2` through its UI or the same `ServerSetting` model save path.
2. Delete only the five new Preview Generator size keys above with direct PHP `occ config:app:delete previewgenerator <key>`; the original backup confirms these keys were unset. Preserve the earlier 60-second budget.
3. Remove only the newly created `/config/nginx/site-confs/upload-timing.conf`, run `nginx -t`, then gracefully reload nginx. The timing log is diagnostic data; existing log rotation handles retention.

References: [Preview Generator size configuration](https://github.com/nextcloud/previewgenerator#available-configuration-options), [Tailscale connection types](https://tailscale.com/docs/reference/connection-types), [nginx request timing variables](https://nginx.org/en/docs/http/ngx_http_log_module.html).

### Additional upload observability and pending health-check plan

- Added privacy-safe PHP-FPM access logging in `/config/php/www2.conf`: timestamp, method, status, elapsed milliseconds, peak memory KiB, and request CPU percentage. Backup: `/config/tuning-backups/20260926-upload-observability/www2.conf`. Native syntax check passed, graceful reload succeeded, and a status GET produced a timing record with HTTP 200. No request URLs, credentials, or client addresses are included. Existing nginx log rotation covers `/config/log/nginx/php-performance.log`.
- Rollback: restore that backup to `/config/php/www2.conf`, run `php-fpm84 -t`, then gracefully reload the master with SIGUSR2. Request wall time includes streamed upload receipt; CPU versus wall timing alone cannot distinguish all network, database, and lock waits.
- Ten further Android Tailscale probes all used DERP Singapore, 126 ms to 1.568 s; no direct route was established. This is current-route evidence, not retrospective proof for the entire earlier batch.
- Prepared five core/proxy interval changes (four Coolify core services at 5 s, proxy at 4 s) to 30 s under `/home/mostak/healthcheck-plan-20260926T0722Z`. Verified nonempty readable original backups. Live core/proxy files remain unchanged pending the AGENTS Tier 2 exact confirmation. Recreating services may briefly interrupt management and ingress. Nextcloud database/Redis 5 s checks require a separate persistent service-definition review.
- Controlled upload plan: same Android client and same ten photos on home Wi-Fi and remote network, recording start/end and route. Inspect request timings, PHP CPU/memory, error logs, and simultaneous host/container samples. Existing logs cannot reconstruct an unrecorded historical CPU peak.

### Latest 21-image test and health-change approval block

- Latest 21 PUTs all returned HTTP 201, completion timestamps 2026-09-26 09:41:46 to 09:45:02 +0200 (13:41:46 to 13:45:02 Bangladesh time). Request bytes total 61,227,010 (includes request overhead), median request duration 6.197 s, range 1.955–24.423 s. Summed PUT duration 176.688 s; first-to-last completion span 196 s is not total batch duration.
- All 21 requests have matching PHP-FPM performance records. Slowest request: elapsed 24.420749 s, CPU 3.85%, approximately 0.9402 CPU-seconds. This supports substantial waiting rather than 24 s of PHP CPU processing; network receipt, I/O/lock waits and scheduling are not fully separated. No simultaneous host samples were recorded for this already-completed upload.
- Build concurrency freshly verified at 1. All inspected containers healthy before attempted changes.
- User requested applying intervals normally and waiving extra backups/checks. Automatic approval review rejected the actual core/proxy apply because the repository Tier 2 workflow still requires exact `CONFIRM EXECUTE`; no live health-check changes executed. Do not bypass the rejection. Five validated changes remain prepared; Nextcloud database/Redis interval persistence remains separately pending.

### Downloader visibility correction

Aria2 live mount and dir match `/data/mostak/files/Media Downloads`. The recent 467,527,619-byte movie was present on disk. Reproducing the previous cron wrapper command yielded zero files; direct PHP yielded three files. `/usr/bin/occ` reconstructs the command with `$*` in a shell, losing argument boundaries for the space-containing path. Replaced only the cron invocation with `docker exec -u abc ... php /app/www/public/occ`, preserving the five-minute interval and flock. Exact command verified three files, zero errors. Read-only database query confirms the recent movie in Nextcloud's file cache; phone UI refresh remains user verification. YAML mount required no change. A separate older movie retains an `.aria2` control file; completion of that older transfer was not verified.

Added `docs/13-tailscale-direct-connection-home-guide.md` with WAN/CGNAT checks, DHCP reservation, targeted UDP forwarding, direct-route verification and rollback.
`n### Preview JPEG quality trial`nAt explicit operator request, changed preview app jpeg_quality from 80 to 70 using direct PHP occ. Read-back verified 70 and status endpoint remained healthy, maintenance false. Applies to newly generated JPEG previews; existing cached previews were not deleted or regenerated, and originals are unchanged. Rollback: set preview jpeg_quality to 80 using the same direct PHP occ command.

### Aria2 resume-file visibility

The recent Alex Rider transfer was verified complete by private authenticated RPC (467527619/467527619 bytes, errorCode 0) and had no control file. The 177-byte `.aria2` belongs to the older A Miracle download, whose completion was not verified. No active/waiting tasks were reported at inspection.

Set Nextcloud `forbidden_filename_extensions` to `[".part", ".filepart", ".aria2"]`, preserving default partial extensions. This is a server-wide filename restriction, so `.aria2` uploads via Nextcloud are also blocked. Aria2's direct filesystem writes remain unaffected. The installed scanner checks storage `verifyPath`, skipping forbidden extensions.

Temporarily moved the specific old control file to the persistent Aria2 config volume, scanned the folder to remove its existing cache entry, then restored the control file to its original path. Repeated scan verified two media files, no added auxiliary entry and zero errors, while the resume file remains on disk. Nextcloud status healthy. No movie or resume state was deleted. Rollback: remove only `.aria2` from the extension list and rescan the folder.

### Preview JPEG quality 60 trial
At explicit operator request, changed preview jpeg_quality from 70 to 60. Direct PHP occ read-back verified 60; status endpoint healthy, maintenance false. Newly generated JPEG previews use the new setting. Pixel targets, existing cached previews and original files unchanged. Rollback: set preview jpeg_quality back to 70.

### Permanent-delete cleanup audit
Live Trash and Versions apps enabled; custom retention keys unset (default auto). Installed Trashbin::delete removes associated versions and trashed original. Preview BackgroundCleanupJob removes previews whose source file IDs no longer exist, normally hourly and TIME_INSENSITIVE. maintenance_window_start=20 (UTC; Bangladesh 02:00–06:00 window). At audit 68 orphan source IDs / 564 preview rows remained; native background-job:execute ran this registered cleanup in 2 seconds, exit 0; read-only recount verified 0 / 0. Existing-source previews retained. No trash emptying, global preview reset, or schedule change performed. Device caches, independent duplicates and backups are outside this server-side deletion guarantee.

### Full preview rebuild requested
Operator explicitly requested clearing old cached previews and regenerating them after quality changes. Native preview:cleanup completed at 2026-09-26 08:45 UTC. preview:generate-all launched without parallel workers, nice priority 15, using current quality 60 and five configured targets. Private progress log: /home/mostak/preview-rebuild-20260926/progress.log. Originals unchanged; Nextcloud status healthy during generation. CLI generate-all is not constrained by the background job's 60-second budget. Completion must be confirmed by REGENERATION_DONE; galleries may regenerate requested variants during the rebuild.

Full rebuild completed successfully: REGENERATION_DONE at 2026-09-26 08:52:02 UTC (14:52 Bangladesh), about 7 minutes after reset began. Read-only preview table count: 468 source file IDs / 2787 preview rows, including internal cached variants beyond the five configured targets. No full permanent-trash purge or cleanup schedule change performed.
