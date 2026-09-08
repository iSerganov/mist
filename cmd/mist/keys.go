package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Keys are stored hex-encoded so they can be inspected and pasted.
const (
	privatePerm os.FileMode = 0o600
	publicPerm  os.FileMode = 0o644
)

func writeKey(path string, key []byte, perm os.FileMode) error {
	if err := os.WriteFile(path, []byte(hex.EncodeToString(key)+"\n"), perm); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func readKey(path string, want int) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key: %w", err)
	}
	key, err := hex.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return nil, fmt.Errorf("%s is not a hex-encoded key", path)
	}
	if len(key) != want {
		return nil, fmt.Errorf("%s holds a %d-byte key, expected %d", path, len(key), want)
	}
	return key, nil
}

// stegoPath derives the output name from the carrier: song.ogg → song.stego.ogg.
func stegoPath(input string) string {
	base := strings.TrimSuffix(input, filepath.Ext(input))
	return base + ".stego.ogg"
}

// keyPaths names the keypair after the file it unlocks.
func keyPaths(output string) (pub, priv string) {
	base := strings.TrimSuffix(output, filepath.Ext(output))
	return base + ".pub", base + ".key"
}
