package store

import (
	"crypto/rand"
	"encoding/hex"
)

func newID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	if prefix == "" {
		return hex.EncodeToString(b)
	}
	return prefix + "_" + hex.EncodeToString(b)
}
