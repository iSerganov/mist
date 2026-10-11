package main

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/iSerganov/mist"
)

// sniffedExtensions maps the content types http.DetectContentType reports to
// the extension a recovered image or audio payload is saved under.
var sniffedExtensions = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/gif":       ".gif",
	"image/webp":      ".webp",
	"image/bmp":       ".bmp",
	"audio/wave":      ".wav",
	"audio/aiff":      ".aiff",
	"audio/mpeg":      ".mp3",
	"audio/midi":      ".mid",
	"application/ogg": ".ogg",
}

// payloadFromFile reads path and picks the payload type from its content, not
// its name: images and audio travel as such, anything else as a named file.
func payloadFromFile(p string) (mist.Payload, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return mist.Payload{}, fmt.Errorf("read payload file: %w", err)
	}
	contentType := http.DetectContentType(data)
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return mist.Image(data), nil
	case strings.HasPrefix(contentType, "audio/") || contentType == "application/ogg":
		return mist.Audio(data), nil
	}
	return mist.File(filepath.Base(p), data)
}

// payloadSummary names a payload for the embed report.
func payloadSummary(p mist.Payload) (size int, kind string) {
	if p.Type == mist.PayloadFile {
		if name, data, err := p.FileName(); err == nil {
			return len(data), "file " + name
		}
	}
	return len(p.Data), map[mist.PayloadType]string{
		mist.PayloadText:  "text",
		mist.PayloadImage: "image",
		mist.PayloadAudio: "audio",
	}[p.Type]
}

// saveRecovered writes a non-text payload into dir and returns where it
// landed. It never replaces an existing file: a taken name gets -1, -2, …
func saveRecovered(dir string, r mist.Result) (string, error) {
	name, data := recoveredName(r)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}
	ext := path.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 0; ; i++ {
		candidate := name
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d%s", stem, i, ext)
		}
		target := filepath.Join(dir, candidate)
		f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("save recovered payload: %w", err)
		}
		_, werr := f.Write(data)
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			return "", fmt.Errorf("save recovered payload: %w", werr)
		}
		return target, nil
	}
}

// recoveredName picks a safe base name for a recovered payload. A name that
// came out of the audio is untrusted: only its last path element is used, and
// an empty or dot name falls back to one derived from the frame and content.
func recoveredName(r mist.Result) (name string, data []byte) {
	data = r.Payload.Data
	if r.Payload.Type == mist.PayloadFile {
		stored, content, err := r.Payload.FileName()
		if err == nil {
			data = content
			name = path.Base(strings.ReplaceAll(stored, `\`, "/"))
		}
	}
	if name == "" || name == "." || name == ".." || name == "/" || strings.ContainsRune(name, 0) {
		ext := sniffedExtensions[strings.SplitN(http.DetectContentType(data), ";", 2)[0]]
		if ext == "" {
			ext = ".bin"
		}
		name = fmt.Sprintf("recovered-%d%s", r.FrameIdx, ext)
	}
	return name, data
}
