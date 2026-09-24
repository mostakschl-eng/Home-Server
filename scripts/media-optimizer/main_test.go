package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestClassificationPreservesSupportedFormatsAndSkipsEditableOriginals(t *testing.T) {
	for name, want := range map[string]fileKind{
		"photo.JPG":  kindJPEG,
		"photo.png":  kindPNG,
		"photo.webp": kindWebP,
		"clip.MP4":   kindVideo,
		"clip.mov":   kindVideo,
		"camera.dng": kindSkip,
		"camera.cr3": kindSkip,
		"design.psd": kindSkip,
		"design.psb": kindSkip,
		"notes.pdf":  kindIgnore,
	} {
		if got := classify(name); got != want {
			t.Errorf("classify(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestQueueRecoveryRequeuesInterruptedStagesAndKeepsPending(t *testing.T) {
	store, err := openStore(filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for i, stage := range []string{"pending", "processing", "validated", "replaced", "scanned"} {
		if err := store.enqueue(item{FileID: int64(i + 1), Path: "folder/file.jpg", ETag: "etag", Stage: stage}); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.recover(); err != nil {
		t.Fatal(err)
	}
	items, err := store.pending()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 5 {
		t.Fatalf("pending rows = %d, want 5", len(items))
	}
	for _, item := range items {
		if item.Stage != "pending" {
			t.Errorf("file %d stage = %q, want pending", item.FileID, item.Stage)
		}
	}
}

func TestRunCheckpointPersistsStartAndSuccessSeparately(t *testing.T) {
	store, err := openStore(filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	started := time.Date(2026, 9, 24, 5, 0, 0, 0, time.UTC)
	if err := store.startRun(started); err != nil {
		t.Fatal(err)
	}
	if err := store.finishRun(started.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	gotStart, gotSuccess, err := store.checkpoints()
	if err != nil {
		t.Fatal(err)
	}
	if !gotStart.Equal(started) || !gotSuccess.Equal(started.Add(time.Hour)) {
		t.Fatalf("checkpoints = %v, %v", gotStart, gotSuccess)
	}
}

func TestRefreshHandlesDeleteAndReuploadWithANewNextcloudFileID(t *testing.T) {
	store, err := openStore(filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.enqueue(item{FileID: 1, Path: "photo.jpg", ETag: "old", ModTime: time.Now(), Stage: "processing"}); err != nil {
		t.Fatal(err)
	}
	if err := store.refresh(1, item{FileID: 2, Path: "photo.jpg", ETag: "new", ModTime: time.Now()}); err != nil {
		t.Fatal(err)
	}
	items, err := store.pending()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].FileID != 2 || items[0].ETag != "new" || items[0].Stage != "pending" {
		t.Fatalf("refreshed queue = %#v", items)
	}
}

func TestLocalFilePathRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if _, err := localFilePath(root, "alice", "../../outside.jpg"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
	got, err := localFilePath(root, "alice", "photos/one.jpg")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "alice", "files", "photos", "one.jpg")
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func TestJPEGMetadataScrubKeepsOrientationAndICCButDropsGPSAndXMP(t *testing.T) {
	input := testJPEGWithMetadata()
	got, err := scrubJPEG(input)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got, []byte("ICC_PROFILE\x00")) {
		t.Fatal("ICC profile was removed")
	}
	if !bytes.Contains(got, []byte("Exif\x00\x00")) {
		t.Fatal("EXIF orientation was removed")
	}
	if bytes.Contains(got, []byte("GPSSECRET")) || bytes.Contains(got, []byte("http://ns.adobe.com/xap/1.0/")) {
		t.Fatal("GPS or XMP metadata remains")
	}
	if !bytes.Contains(got, []byte{0x12, 0x01, 0x03, 0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x00}) {
		t.Fatal("EXIF orientation value was not retained")
	}
}

func TestWebDAVDiscoveryRequiresTagsAndQueuesOnlyUntaggedSupportedMedia(t *testing.T) {
	date := "Wed, 24 Sep 2026 00:00:00 GMT"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PROPFIND" {
			t.Fatalf("method = %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/xml")
		switch r.URL.Path {
		case "/remote.php/dav/systemtags/":
			fmt.Fprint(w, `<d:multistatus xmlns:d="DAV:" xmlns:nc="http://nextcloud.org/ns"><d:response><d:propstat><d:prop><nc:id>11</nc:id><nc:display-name>opt</nc:display-name></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response><d:response><d:propstat><d:prop><nc:id>12</nc:id><nc:display-name>skip</nc:display-name></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response></d:multistatus>`)
		case "/remote.php/dav/files/alice/":
			fmt.Fprintf(w, `<d:multistatus xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns" xmlns:nc="http://nextcloud.org/ns"><d:response><d:href>/remote.php/dav/files/alice/photo%%20one.jpg</d:href><d:propstat><d:prop><d:getetag>"e4"</d:getetag><d:getlastmodified>%s</d:getlastmodified><d:getcontentlength>500</d:getcontentlength><oc:fileid>4</oc:fileid><oc:tags/><nc:lock>0</nc:lock></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response></d:multistatus>`, date)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.Error(w, "unexpected", 404)
		}
	}))
	defer srv.Close()
	c := config{baseURL: srv.URL, username: "alice", password: "test", optTag: "opt", skipTag: "skip"}
	n := newDAV(c)
	n.client = srv.Client()
	if err := n.connect(t.Context()); err != nil {
		t.Fatal(err)
	}
	store, err := openStore(filepath.Join(t.TempDir(), "queue.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := n.discover(t.Context(), store); err != nil {
		t.Fatal(err)
	}
	items, err := store.pending()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].FileID != 4 || items[0].Path != "photo one.jpg" || items[0].ETag != "e4" {
		t.Fatalf("discovered candidates = %#v", items)
	}
}

func TestWebDAVCurrentItemRejectsReportedLock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<d:multistatus xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns" xmlns:nc="http://nextcloud.org/ns"><d:response><d:propstat><d:prop><d:getetag>e</d:getetag><d:getlastmodified>Wed, 24 Sep 2026 00:00:00 GMT</d:getlastmodified><d:getcontentlength>1</d:getcontentlength><oc:fileid>7</oc:fileid><nc:lock>1</nc:lock></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response></d:multistatus>`)
	}))
	defer srv.Close()
	c := config{baseURL: srv.URL, username: "alice"}
	n := newDAV(c)
	n.client = srv.Client()
	_, err := n.currentItem(t.Context(), item{FileID: 7, Path: "photo.jpg"})
	if err != errFileLocked {
		t.Fatalf("lock result = %v, want errFileLocked", err)
	}
}

func TestWebDAVCurrentItemReadsSystemTagsForCrashRecovery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<d:multistatus xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns" xmlns:nc="http://nextcloud.org/ns"><d:response><d:propstat><d:prop><d:getetag>e</d:getetag><d:getlastmodified>Wed, 24 Sep 2026 00:00:00 GMT</d:getlastmodified><d:getcontentlength>1</d:getcontentlength><oc:fileid>7</oc:fileid><nc:lock>0</nc:lock><oc:tags><oc:tag>opt</oc:tag></oc:tags></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response></d:multistatus>`)
	}))
	defer srv.Close()
	c := config{baseURL: srv.URL, username: "alice"}
	n := newDAV(c)
	n.client = srv.Client()
	got, err := n.currentItem(t.Context(), item{FileID: 7, Path: "photo.jpg"})
	if err != nil {
		t.Fatal(err)
	}
	if !tagged(got.Tags, "opt") {
		t.Fatalf("current Nextcloud tags = %#v, want opt", got.Tags)
	}
}

func testJPEGWithMetadata() []byte {
	segment := func(marker byte, payload []byte) []byte {
		out := []byte{0xff, marker, byte((len(payload) + 2) >> 8), byte(len(payload) + 2)}
		return append(out, payload...)
	}
	tiff := make([]byte, 8+2+2*12+4+12)
	copy(tiff, []byte{'I', 'I'})
	binary.LittleEndian.PutUint16(tiff[2:4], 42)
	binary.LittleEndian.PutUint32(tiff[4:8], 8)
	binary.LittleEndian.PutUint16(tiff[8:10], 2)
	p := 10
	binary.LittleEndian.PutUint16(tiff[p:p+2], 0x0112)
	binary.LittleEndian.PutUint16(tiff[p+2:p+4], 3)
	binary.LittleEndian.PutUint32(tiff[p+4:p+8], 1)
	binary.LittleEndian.PutUint16(tiff[p+8:p+10], 6)
	p += 12
	binary.LittleEndian.PutUint16(tiff[p:p+2], 0x8825)
	binary.LittleEndian.PutUint16(tiff[p+2:p+4], 4)
	binary.LittleEndian.PutUint32(tiff[p+4:p+8], 1)
	binary.LittleEndian.PutUint32(tiff[p+8:p+12], 38)
	copy(tiff[38:], []byte("GPSSECRET"))
	exif := append([]byte("Exif\x00\x00"), tiff...)
	icc := append([]byte("ICC_PROFILE\x00"), 1, 1, 1, 2, 3)
	return bytes.Join([][]byte{{0xff, 0xd8}, segment(0xe1, exif), segment(0xe1, []byte("http://ns.adobe.com/xap/1.0/\x00xmp")), segment(0xe2, icc), {0xff, 0xda, 0, 2, 1, 2, 3, 0xff, 0xd9}}, nil)
}
