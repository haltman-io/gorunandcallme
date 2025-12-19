package util

import (
	"crypto/rand"
	"encoding/hex"
)

func NewID(prefix string) string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	id := hex.EncodeToString(b[:])
	if prefix == "" {
		return id
	}
	return prefix + "_" + id
}
