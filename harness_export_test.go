//go:build harness

package mist

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// exportChunks is how many chunks of each carrier go to the external CNN
// warden. It trains on these, so more costs time and gains little.
const exportChunks = 16

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// exportCarrier writes the values both copies of a carrier give the
// detectors, for tools/cnn_warden. It does nothing unless
// MIST_HARNESS_EXPORT names a directory.
func exportCarrier(format string, c harnessCarrier, clean, stego []int32) error {
	root := os.Getenv("MIST_HARNESS_EXPORT")
	if root == "" {
		return nil
	}
	dir := filepath.Join(root, strings.ReplaceAll(format, "/", "-"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	stem := filepath.Join(dir, unsafeName.ReplaceAllString(c.name, "_"))
	for suffix, vals := range map[string][]int32{".clean.i32": clean, ".stego.i32": stego} {
		vals = vals[:min(len(vals), exportChunks*chunkValues)]
		buf := make([]byte, 4*len(vals))
		for i, v := range vals {
			binary.LittleEndian.PutUint32(buf[4*i:], uint32(v))
		}
		if err := os.WriteFile(stem+suffix, buf, 0o644); err != nil {
			return err
		}
	}
	meta, err := json.Marshal(map[string]string{
		"name": c.name, "category": c.category, "lineage": c.lineage, "license": c.license,
	})
	if err != nil {
		return err
	}
	return os.WriteFile(stem+".json", meta, 0o644)
}
