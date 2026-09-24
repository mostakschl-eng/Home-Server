package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type config struct {
	baseURL, username, password, dataDir, database, container, occPath            string
	optTag, skipTag, jpegtran, oxipng, webpmux, ffmpeg, ffprobe, docker, execUser string
	stableFor                                                                     time.Duration
}

func loadConfig() (config, error) {
	duration, err := time.ParseDuration(env("STABLE_FOR", "3m"))
	if err != nil || duration < 0 {
		return config{}, errors.New("STABLE_FOR must be a non-negative duration such as 3m")
	}
	c := config{
		baseURL:  strings.TrimRight(os.Getenv("NEXTCLOUD_URL"), "/"),
		username: os.Getenv("NEXTCLOUD_USERNAME"), password: os.Getenv("NEXTCLOUD_APP_PASSWORD"),
		dataDir: os.Getenv("NEXTCLOUD_DATA_DIR"), database: env("QUEUE_DB", "/var/lib/media-optimizer/queue.sqlite"),
		container: os.Getenv("NEXTCLOUD_CONTAINER"), occPath: env("NEXTCLOUD_OCC_PATH", "/config/www/nextcloud/occ"),
		optTag: env("OPT_TAG", "opt"), skipTag: env("SKIP_TAG", "skip"),
		jpegtran: env("JPEGTRAN", "jpegtran"), oxipng: env("OXIPNG", "oxipng"),
		webpmux: env("WEBPMUX", "webpmux"),
		ffmpeg:  env("FFMPEG", "ffmpeg"), ffprobe: env("FFPROBE", "ffprobe"), docker: env("DOCKER", "docker"), execUser: env("NEXTCLOUD_EXEC_USER", "abc"), stableFor: duration,
	}
	if c.baseURL == "" || c.username == "" || c.password == "" || c.dataDir == "" || c.container == "" {
		return c, errors.New("required settings missing: NEXTCLOUD_URL, NEXTCLOUD_USERNAME, NEXTCLOUD_APP_PASSWORD, NEXTCLOUD_DATA_DIR, NEXTCLOUD_CONTAINER")
	}
	if strings.EqualFold(c.optTag, c.skipTag) {
		return c, errors.New("OPT_TAG and SKIP_TAG must be different tag names")
	}
	return c, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	check := flag.Bool("check-config", false, "validate required environment settings without contacting Nextcloud")
	flag.Parse()
	c, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}
	if *check {
		fmt.Println("Required settings are present. Server connectivity was not tested.")
		return
	}
	if flag.NArg() != 1 || flag.Arg(0) != "run" {
		log.Fatal("usage: media-optimizer [-check-config] run")
	}
	if err := os.MkdirAll(filepath.Dir(c.database), 0o700); err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	lock, err := acquireLock(c.database + ".lock")
	if err != nil {
		log.Fatal(err)
	}
	defer lock.Close()
	if err := run(ctx, c); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, c config) (runErr error) {
	if err := os.MkdirAll(filepath.Dir(c.database), 0o700); err != nil {
		return fmt.Errorf("create database directory: %w", err)
	}
	store, err := openStore(c.database)
	if err != nil {
		return err
	}
	defer store.Close()
	started := time.Now().UTC()
	if err := store.startRun(started); err != nil {
		return err
	}
	defer func() {
		_ = store.setState("last_run_finished_at", time.Now().UTC().Format(time.RFC3339Nano))
		if runErr != nil {
			_ = store.setState("last_run_status", "failed")
		}
	}()
	dav := newDAV(c)
	if err := dav.connect(ctx); err != nil {
		return err
	}
	if err := checkRuntimeTools(c); err != nil {
		return err
	}
	if err := store.recover(); err != nil {
		return err
	}
	discovered := false
	for {
		items, err := store.pending()
		if err != nil {
			return err
		}
		if len(items) == 0 {
			if discovered {
				break
			}
			if err := dav.discover(ctx, store); err != nil {
				return err
			}
			discovered = true
			items, err = store.pending()
			if err != nil {
				return err
			}
			if len(items) == 0 {
				break
			}
		}
		if err := processOne(ctx, c, dav, store, items[0]); err != nil {
			return fmt.Errorf("file %s: %w", items[0].Path, err)
		}
	}
	remaining, err := store.count()
	if err != nil {
		return err
	}
	if remaining > 0 {
		return fmt.Errorf("run ended with %d deferred or unfinished file(s); they will be retried next run", remaining)
	}
	if err := store.finishRun(time.Now().UTC()); err != nil {
		return err
	}
	log.Printf("run finished; last successful run recorded in %s", c.database)
	return nil
}

func checkRuntimeTools(c config) error {
	for name, path := range map[string]string{"jpegtran": c.jpegtran, "oxipng": c.oxipng, "webpmux": c.webpmux, "ffmpeg": c.ffmpeg, "ffprobe": c.ffprobe, "docker": c.docker, "chown": "chown"} {
		if _, err := exec.LookPath(path); err != nil {
			return fmt.Errorf("required tool %s not found: %w", name, err)
		}
	}
	return nil
}

func processOne(ctx context.Context, c config, dav *nextcloud, store *queueStore, item item) error {
	kind := classify(item.Path)
	if kind == kindIgnore {
		return store.remove(item.FileID)
	}
	latest, err := dav.currentItem(ctx, item)
	if errors.Is(err, errFileLocked) {
		return store.setStage(item.FileID, "deferred")
	}
	if errors.Is(err, errRemoteNotFound) {
		return store.remove(item.FileID)
	}
	if err != nil {
		return err
	}
	if tagged(latest.Tags, dav.c.optTag) || tagged(latest.Tags, dav.c.skipTag) {
		return store.remove(item.FileID)
	}
	if !sameRemoteVersion(item, latest) {
		if err := store.refresh(item.FileID, latest); err != nil {
			return err
		}
		return nil
	}
	if kind == kindSkip {
		if err := dav.addTag(ctx, latest.FileID, dav.skipTagID); err != nil {
			return err
		}
		return store.remove(item.FileID)
	}
	local, err := localFilePath(c.dataDir, c.username, item.Path)
	if err != nil {
		return err
	}
	if err := ensureRegularNoSymlinks(filepath.Join(c.dataDir, c.username, "files"), local); err != nil {
		return err
	}
	if err := store.setStage(item.FileID, "processing"); err != nil {
		return err
	}
	inputInfo, err := os.Stat(local)
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}
	wait := c.stableFor - time.Since(inputInfo.ModTime())
	if wait > c.stableFor {
		wait = c.stableFor
	}
	if wait > 0 {
		log.Printf("waiting %s for upload stability: %s", wait.Round(time.Second), item.Path)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	current, err := os.Stat(local)
	if err != nil {
		return err
	}
	if !sameFileState(inputInfo, current) {
		return errors.New("file changed during stability wait; retry next run")
	}
	latest, err = dav.currentItem(ctx, item)
	if errors.Is(err, errFileLocked) {
		return store.setStage(item.FileID, "deferred")
	}
	if errors.Is(err, errRemoteNotFound) {
		return store.remove(item.FileID)
	}
	if err != nil {
		return err
	}
	if tagged(latest.Tags, dav.c.optTag) || tagged(latest.Tags, dav.c.skipTag) {
		return store.remove(item.FileID)
	}
	if !sameRemoteVersion(item, latest) {
		if err := store.refresh(item.FileID, latest); err != nil {
			return err
		}
		return nil
	}
	tmp, err := os.CreateTemp(filepath.Dir(local), ".media-optimizer-*"+filepath.Ext(local))
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	defer os.Remove(tmpPath)
	if err := optimizeFile(ctx, c, kind, local, tmpPath); err != nil {
		return err
	}
	if err := validateOutput(ctx, c, kind, local, tmpPath); err != nil {
		return err
	}
	if err := store.setStage(item.FileID, "validated"); err != nil {
		return err
	}
	outputInfo, err := os.Stat(tmpPath)
	if err != nil || outputInfo.Size() == 0 {
		return errors.New("optimizer produced an empty output")
	}
	latest, err = dav.currentItem(ctx, item)
	if errors.Is(err, errFileLocked) {
		return store.setStage(item.FileID, "deferred")
	}
	if errors.Is(err, errRemoteNotFound) {
		return store.remove(item.FileID)
	}
	if err != nil {
		return err
	}
	if tagged(latest.Tags, dav.c.optTag) || tagged(latest.Tags, dav.c.skipTag) {
		return store.remove(item.FileID)
	}
	if !sameRemoteVersion(item, latest) {
		if err := store.refresh(item.FileID, latest); err != nil {
			return err
		}
		return nil
	}
	if outputInfo.Size() < inputInfo.Size() {
		latest, err := os.Stat(local)
		if err != nil || !sameFileState(inputInfo, latest) {
			return errors.New("source changed before replacement; output discarded")
		}
		if err := os.Chmod(tmpPath, inputInfo.Mode()); err != nil {
			return fmt.Errorf("preserve source permissions: %w", err)
		}
		if err := os.Chtimes(tmpPath, inputInfo.ModTime(), inputInfo.ModTime()); err != nil {
			return fmt.Errorf("preserve source modification time: %w", err)
		}
		if err := exec.Command("chown", "--reference="+local, tmpPath).Run(); err != nil {
			return fmt.Errorf("preserve source owner (run as the Nextcloud data owner or install chown): %w", err)
		}
		if err := replaceFile(tmpPath, local); err != nil {
			return fmt.Errorf("atomic replacement: %w", err)
		}
		log.Printf("optimized %s: %.2f MiB -> %.2f MiB (%.1f%% saved)", item.Path, float64(inputInfo.Size())/1048576, float64(outputInfo.Size())/1048576, 100*(1-float64(outputInfo.Size())/float64(inputInfo.Size())))
	} else {
		log.Printf("no size reduction; leaving original unchanged: %s", item.Path)
	}
	if err := store.setStage(item.FileID, "replaced"); err != nil {
		return err
	}
	if err := dav.scan(ctx, item.Path); err != nil {
		return err
	}
	if err := store.setStage(item.FileID, "scanned"); err != nil {
		return err
	}
	if err := dav.addTag(ctx, item.FileID, dav.optTagID); err != nil {
		return err
	}
	if err := store.setStage(item.FileID, "tagged"); err != nil {
		return err
	}
	return store.remove(item.FileID)
}

func tagged(tags []string, name string) bool {
	for _, tag := range tags {
		if strings.EqualFold(strings.TrimSpace(tag), name) {
			return true
		}
	}
	return false
}

func localFilePath(root, username, rel string) (string, error) {
	if username == "." || username == ".." || filepath.Base(username) != username {
		return "", errors.New("invalid Nextcloud username for local path")
	}
	if filepath.IsAbs(rel) || strings.Contains(rel, "\\") {
		return "", errors.New("invalid Nextcloud relative path")
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("Nextcloud path escapes the user files directory")
	}
	base := filepath.Join(root, username, "files")
	full := filepath.Join(base, clean)
	check, err := filepath.Rel(base, full)
	if err != nil || check == ".." || strings.HasPrefix(check, ".."+string(filepath.Separator)) {
		return "", errors.New("Nextcloud path escapes the user files directory")
	}
	return full, nil
}

func ensureRegularNoSymlinks(base, full string) error {
	baseInfo, err := os.Lstat(base)
	if err != nil {
		return err
	}
	if baseInfo.Mode()&os.ModeSymlink != 0 || !baseInfo.IsDir() {
		return errors.New("user files root is not a real directory")
	}
	rel, err := filepath.Rel(base, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("media path escapes user files directory")
	}
	current := base
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, e := os.Lstat(current)
		if e != nil {
			return e
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("refusing to process symlinks in Nextcloud storage")
		}
		if current == full && !info.Mode().IsRegular() {
			return errors.New("media is not a regular file")
		}
	}
	return nil
}

func sameFileState(a, b os.FileInfo) bool {
	return a.Size() == b.Size() && a.ModTime().Equal(b.ModTime())
}
func sameRemoteVersion(a, b item) bool {
	return a.ETag == b.ETag && a.Size == b.Size && a.ModTime.Equal(b.ModTime)
}
