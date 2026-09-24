package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type fileKind string

const (
	kindJPEG   fileKind = "jpeg"
	kindPNG    fileKind = "png"
	kindWebP   fileKind = "webp"
	kindVideo  fileKind = "video"
	kindSkip   fileKind = "skip"
	kindIgnore fileKind = "ignore"
)

func classify(name string) fileKind {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg":
		return kindJPEG
	case ".png":
		return kindPNG
	case ".webp":
		return kindWebP
	case ".mp4", ".mov", ".m4v", ".mkv", ".webm", ".avi":
		return kindVideo
	case ".3fr", ".ari", ".arw", ".bay", ".cap", ".cr2", ".cr3", ".crw", ".dcs", ".dcr", ".dng", ".drf", ".eip", ".erf", ".fff", ".gpr", ".iiq", ".k25", ".kdc", ".mdc", ".mef", ".mos", ".mrw", ".nef", ".nrw", ".obm", ".orf", ".pef", ".ptx", ".pxn", ".r3d", ".raf", ".raw", ".rwl", ".rw2", ".rwz", ".sr2", ".srf", ".srw", ".x3f", ".psd", ".psb":
		return kindSkip
	default:
		return kindIgnore
	}
}

func optimizeFile(ctx context.Context, c config, kind fileKind, input, output string) error {
	switch kind {
	case kindJPEG:
		if err := runCommand(ctx, c.jpegtran, "-copy", "all", "-optimize", "-progressive", "-outfile", output, input); err != nil {
			return err
		}
		return scrubJPEGFile(output)
	case kindPNG:
		return runCommand(ctx, c.oxipng, "-o", "4", "--strip", "safe", "--out", output, input)
	case kindWebP:
		stage := output + ".stripped-exif"
		defer os.Remove(stage)
		if err := runCommand(ctx, c.webpmux, "-strip", "exif", input, "-o", stage); err != nil {
			return err
		}
		return runCommand(ctx, c.webpmux, "-strip", "xmp", stage, "-o", output)
	case kindVideo:
		return runCommand(ctx, c.ffmpeg, "-hide_banner", "-loglevel", "error", "-nostdin", "-y", "-i", input, "-map", "0", "-map_metadata", "-1", "-map_metadata:s", "-1", "-map_chapters", "-1", "-c", "copy", output)
	default:
		return fmt.Errorf("unsupported media kind %q", kind)
	}
}

func validateOutput(ctx context.Context, c config, kind fileKind, original, output string) error {
	info, err := os.Stat(output)
	if err != nil {
		return fmt.Errorf("read optimized output: %w", err)
	}
	if info.Size() == 0 {
		return errors.New("optimized output is empty")
	}
	args := []string{"-hide_banner", "-loglevel", "error", "-xerror", "-nostdin", "-i", output, "-f", "null", "-"}
	if err := runCommand(ctx, c.ffmpeg, args...); err != nil {
		return fmt.Errorf("decode validation: %w", err)
	}
	if kind == kindJPEG || kind == kindPNG || kind == kindWebP {
		before, e := imageDimensions(ctx, c.ffprobe, original)
		if e != nil {
			return e
		}
		after, e := imageDimensions(ctx, c.ffprobe, output)
		if e != nil {
			return e
		}
		if before != after {
			return fmt.Errorf("image dimensions changed from %s to %s", before, after)
		}
	}
	return nil
}

func imageDimensions(ctx context.Context, ffprobe, path string) (string, error) {
	out, err := exec.CommandContext(ctx, ffprobe, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=s=x:p=0", path).Output()
	if err != nil {
		return "", fmt.Errorf("read image dimensions: %w", err)
	}
	d := strings.TrimSpace(string(out))
	if d == "" || strings.Contains(d, "N/A") {
		return "", errors.New("media has no readable video/image dimensions")
	}
	return d, nil
}
func runCommand(ctx context.Context, command string, args ...string) error {
	out, err := exec.CommandContext(ctx, command, args...).CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if detail != "" {
			return fmt.Errorf("%s: %w: %s", filepath.Base(command), err, detail)
		}
		return fmt.Errorf("%s: %w", filepath.Base(command), err)
	}
	return nil
}

// JPEG markers retained here are required for normal display or color: JFIF,
// Adobe transform, ICC profile, and EXIF orientation. Other metadata is dropped.
func scrubJPEGFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	cleaned, err := scrubJPEG(b)
	if err != nil {
		return err
	}
	return os.WriteFile(path, cleaned, 0o600)
}

func scrubJPEG(b []byte) ([]byte, error) {
	if len(b) < 4 || b[0] != 0xff || b[1] != 0xd8 {
		return nil, errors.New("invalid JPEG header")
	}
	out := append([]byte{}, b[:2]...)
	pos := 2
	for pos < len(b) {
		start := pos
		if b[pos] != 0xff {
			return nil, errors.New("invalid JPEG marker stream")
		}
		for pos < len(b) && b[pos] == 0xff {
			pos++
		}
		if pos >= len(b) {
			return nil, io.ErrUnexpectedEOF
		}
		marker := b[pos]
		pos++
		if marker == 0xda {
			if pos+2 > len(b) {
				return nil, io.ErrUnexpectedEOF
			}
			n := int(binary.BigEndian.Uint16(b[pos : pos+2]))
			end := pos + n
			if n < 2 || end > len(b) {
				return nil, errors.New("invalid JPEG scan header")
			}
			out = append(out, b[start:]...)
			return out, nil
		}
		if marker == 0xd9 || (marker >= 0xd0 && marker <= 0xd7) || marker == 0x01 {
			out = append(out, b[start:pos]...)
			continue
		}
		if pos+2 > len(b) {
			return nil, io.ErrUnexpectedEOF
		}
		n := int(binary.BigEndian.Uint16(b[pos : pos+2]))
		end := pos + n
		if n < 2 || end > len(b) {
			return nil, errors.New("invalid JPEG segment length")
		}
		payload := b[pos+2 : end]
		keep := false
		var replacement []byte
		switch marker {
		case 0xe0:
			keep = bytes.HasPrefix(payload, []byte("JFIF\x00")) || bytes.HasPrefix(payload, []byte("JFXX\x00"))
		case 0xee:
			keep = bytes.HasPrefix(payload, []byte("Adobe"))
		case 0xe2:
			keep = bytes.HasPrefix(payload, []byte("ICC_PROFILE\x00"))
		case 0xe1:
			if bytes.HasPrefix(payload, []byte("Exif\x00\x00")) {
				replacement = orientationEXIF(payload)
				keep = len(replacement) > 0
			}
		}
		if keep {
			if replacement != nil {
				out = appendSegment(out, marker, replacement)
			} else {
				out = append(out, b[start:end]...)
			}
		}
		pos = end
	}
	return nil, errors.New("JPEG has no image scan")
}

func appendSegment(dst []byte, marker byte, payload []byte) []byte {
	dst = append(dst, 0xff, marker)
	n := len(payload) + 2
	dst = append(dst, byte(n>>8), byte(n))
	return append(dst, payload...)
}
func orientationEXIF(payload []byte) []byte {
	t := payload[6:]
	if len(t) < 8 {
		return nil
	}
	var order binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return nil
	}
	if order.Uint16(t[2:4]) != 42 {
		return nil
	}
	off := int(order.Uint32(t[4:8]))
	if off < 8 || off+2 > len(t) {
		return nil
	}
	count := int(order.Uint16(t[off : off+2]))
	if count > 4096 || off+2+count*12+4 > len(t) {
		return nil
	}
	for i := 0; i < count; i++ {
		p := off + 2 + i*12
		tag := order.Uint16(t[p : p+2])
		typ := order.Uint16(t[p+2 : p+4])
		n := order.Uint32(t[p+4 : p+8])
		if tag != 0x0112 || typ != 3 || n != 1 {
			continue
		}
		value := order.Uint16(t[p+8 : p+10])
		minimal := make([]byte, 26)
		copy(minimal, t[:2])
		order.PutUint16(minimal[2:4], 42)
		order.PutUint32(minimal[4:8], 8)
		order.PutUint16(minimal[8:10], 1)
		order.PutUint16(minimal[10:12], 0x0112)
		order.PutUint16(minimal[12:14], 3)
		order.PutUint32(minimal[14:18], 1)
		order.PutUint16(minimal[18:20], value)
		return append([]byte("Exif\x00\x00"), minimal...)
	}
	return nil
}

func replaceFile(from, to string) error { return os.Rename(from, to) }
