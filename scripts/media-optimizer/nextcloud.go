package main

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var errRemoteNotFound = errors.New("Nextcloud file no longer exists")
var errFileLocked = errors.New("Nextcloud reports the file is locked")

type nextcloud struct {
	c                   config
	client              *http.Client
	filesRoot           *url.URL
	optTagID, skipTagID string
}

type davProp struct {
	ResourceType struct {
		Collection *struct{} `xml:"collection"`
	} `xml:"resourcetype"`
	ETag     string  `xml:"getetag"`
	Modified string  `xml:"getlastmodified"`
	Size     int64   `xml:"getcontentlength"`
	FileID   string  `xml:"fileid"`
	Lock     *string `xml:"lock"`
	Tags     *struct {
		Tag []string `xml:"tag"`
	} `xml:"tags"`
	TagID       string `xml:"id"`
	TagName     string `xml:"display-name"`
	DisplayName string `xml:"displayname"`
	Name        string `xml:"name"`
}
type davPropstat struct {
	Status string  `xml:"status"`
	Prop   davProp `xml:"prop"`
}
type davResponse struct {
	Href      string        `xml:"href"`
	Propstats []davPropstat `xml:"propstat"`
}
type davMultiStatus struct {
	Responses []davResponse `xml:"response"`
}

func newDAV(c config) *nextcloud {
	u, _ := url.Parse(c.baseURL)
	u.Path = strings.TrimSuffix(u.Path, "/") + "/remote.php/dav/files/" + c.username + "/"
	return &nextcloud{c: c, client: &http.Client{Timeout: 2 * time.Minute}, filesRoot: u}
}

func (n *nextcloud) davRootURL() url.URL {
	u := *n.filesRoot
	u.Path = strings.TrimSuffix(n.filesRoot.Path, "files/"+n.c.username+"/")
	return u
}

func (n *nextcloud) connect(ctx context.Context) error {
	u := n.davRootURL()
	u.Path += "systemtags/"
	responses, err := n.propfind(ctx, &u, "1", `<d:prop><d:resourcetype/><nc:id/><nc:display-name/><d:displayname/></d:prop>`)
	if err != nil {
		return fmt.Errorf("list Nextcloud system tags: %w", err)
	}
	for _, r := range responses {
		p, ok := successfulProp(r)
		if !ok || p.TagID == "" {
			continue
		}
		name := p.TagName
		if name == "" {
			name = p.DisplayName
		}
		if name == "" {
			name = p.Name
		}
		if strings.EqualFold(name, n.c.optTag) {
			if n.optTagID != "" && n.optTagID != p.TagID {
				return fmt.Errorf("multiple Nextcloud system tags named %q; use unique tag names", n.c.optTag)
			}
			n.optTagID = p.TagID
		}
		if strings.EqualFold(name, n.c.skipTag) {
			if n.skipTagID != "" && n.skipTagID != p.TagID {
				return fmt.Errorf("multiple Nextcloud system tags named %q; use unique tag names", n.c.skipTag)
			}
			n.skipTagID = p.TagID
		}
	}
	if n.optTagID == "" || n.skipTagID == "" {
		return fmt.Errorf("required system tags %q and %q were not both found; create them in Nextcloud first", n.c.optTag, n.c.skipTag)
	}
	return nil
}

func successfulProp(r davResponse) (davProp, bool) {
	for _, ps := range r.Propstats {
		if strings.Contains(ps.Status, " 200 ") {
			return ps.Prop, true
		}
	}
	return davProp{}, false
}

func (n *nextcloud) propfind(ctx context.Context, u *url.URL, depth, body string) ([]davResponse, error) {
	xmlBody := `<?xml version="1.0"?><d:propfind xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns" xmlns:nc="http://nextcloud.org/ns">` + body + `</d:propfind>`
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", u.String(), strings.NewReader(xmlBody))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(n.c.username, n.c.password)
	req.Header.Set("Depth", depth)
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	resp, err := n.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, errRemoteNotFound
	}
	if resp.StatusCode != 207 && resp.StatusCode != 200 {
		return nil, fmt.Errorf("PROPFIND returned HTTP %s", resp.Status)
	}
	var result davMultiStatus
	if err := xml.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode WebDAV response: %w", err)
	}
	return result.Responses, nil
}

func (n *nextcloud) discover(ctx context.Context, s *queueStore) error {
	if n.optTagID == "" || n.skipTagID == "" {
		return errors.New("Nextcloud tag discovery was not initialized")
	}
	visited := map[string]bool{}
	var walk func(*url.URL) error
	walk = func(dir *url.URL) error {
		if visited[dir.String()] {
			return nil
		}
		visited[dir.String()] = true
		responses, err := n.propfind(ctx, dir, "1", `<d:prop><d:resourcetype/><d:getetag/><d:getlastmodified/><d:getcontentlength/><oc:fileid/><oc:tags/><nc:lock/></d:prop>`)
		if err != nil {
			return err
		}
		for _, r := range responses {
			p, ok := successfulProp(r)
			if !ok {
				continue
			}
			href, err := url.Parse(r.Href)
			if err != nil {
				return err
			}
			basePath := strings.TrimSuffix(n.filesRoot.Path, "/")
			if href.Path != basePath && !strings.HasPrefix(href.Path, basePath+"/") {
				continue
			}
			rel := strings.TrimPrefix(strings.TrimPrefix(href.Path, basePath), "/")
			rel = strings.TrimSuffix(rel, "/")
			if rel == "" {
				continue
			}
			if p.ResourceType.Collection != nil {
				child := *n.filesRoot
				child.Path = strings.TrimSuffix(n.filesRoot.Path, "/") + "/" + rel + "/"
				if err := walk(&child); err != nil {
					return err
				}
				continue
			}
			if p.FileID == "" {
				return fmt.Errorf("WebDAV omitted Nextcloud file id for %s", rel)
			}
			if p.Tags == nil {
				return fmt.Errorf("WebDAV omitted oc:tags for %s; refusing to guess whether opt/skip tags are absent", rel)
			}
			hasOpt, hasSkip := false, false
			for _, tag := range p.Tags.Tag {
				tag = strings.TrimSpace(tag)
				hasOpt = hasOpt || strings.EqualFold(tag, n.c.optTag)
				hasSkip = hasSkip || strings.EqualFold(tag, n.c.skipTag)
			}
			if hasOpt || hasSkip || classify(rel) == kindIgnore {
				continue
			}
			if p.Lock == nil {
				return errors.New("Nextcloud omitted the nc:lock property; refusing discovery without lock status")
			}
			if *p.Lock == "1" {
				continue
			}
			mtime, err := http.ParseTime(strings.TrimSpace(p.Modified))
			if err != nil {
				return fmt.Errorf("invalid last-modified value for %s: %w", rel, err)
			}
			id, err := strconv.ParseInt(p.FileID, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid Nextcloud file id %q: %w", p.FileID, err)
			}
			if err := s.enqueue(item{FileID: id, Path: rel, ETag: strings.Trim(p.ETag, `"`), Size: p.Size, ModTime: mtime}); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(n.filesRoot)
}

func (n *nextcloud) fileURL(rel string) (*url.URL, error) {
	u := *n.filesRoot
	if strings.Contains(rel, `\`) || path.IsAbs(rel) || path.Clean(rel) == ".." || strings.HasPrefix(path.Clean(rel), "../") {
		return nil, errors.New("unsafe WebDAV path")
	}
	u.Path = strings.TrimSuffix(n.filesRoot.Path, "/") + "/" + rel
	return &u, nil
}

func (n *nextcloud) currentItem(ctx context.Context, old item) (item, error) {
	u, err := n.fileURL(old.Path)
	if err != nil {
		return item{}, err
	}
	responses, err := n.propfind(ctx, u, "0", `<d:prop><d:getetag/><d:getlastmodified/><d:getcontentlength/><oc:fileid/><nc:lock/><oc:tags/></d:prop>`)
	if err != nil {
		return item{}, fmt.Errorf("check current file version: %w", err)
	}
	for _, r := range responses {
		p, ok := successfulProp(r)
		if !ok {
			continue
		}
		if p.Lock == nil {
			return item{}, errors.New("Nextcloud omitted the nc:lock property; refusing direct file replacement")
		}
		if *p.Lock == "1" {
			return item{}, errFileLocked
		}
		if p.Tags == nil {
			return item{}, errors.New("Nextcloud omitted the oc:tags property; cannot resume idempotently")
		}
		mt, e := http.ParseTime(strings.TrimSpace(p.Modified))
		if e != nil {
			return item{}, e
		}
		id, e := strconv.ParseInt(p.FileID, 10, 64)
		if e != nil {
			return item{}, e
		}
		return item{FileID: id, Path: old.Path, ETag: strings.Trim(p.ETag, `"`), Size: p.Size, ModTime: mt, Stage: "pending", Tags: p.Tags.Tag}, nil
	}
	return item{}, errors.New("Nextcloud returned no file metadata")
}

func (n *nextcloud) addTag(ctx context.Context, fileID int64, tagID string) error {
	u := n.davRootURL()
	u.Path += "systemtags-relations/files/" + strconv.FormatInt(fileID, 10) + "/" + url.PathEscape(tagID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(n.c.username, n.c.password)
	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("assign Nextcloud tag returned HTTP %s", resp.Status)
	}
	return nil
}

func (n *nextcloud) scan(ctx context.Context, rel string) error {
	args := []string{"exec", "-u", n.c.execUser, n.c.container, "php", n.c.occPath, "files:scan", "--path", n.c.username + "/files/" + filepath.ToSlash(rel)}
	return runCommand(ctx, n.c.docker, args...)
}
