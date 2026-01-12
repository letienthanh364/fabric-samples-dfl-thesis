package common

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// GeneratePrefixedID produces a globally unique identifier with the supplied prefix.
func GeneratePrefixedID(prefix string) string {
	prefix = strings.TrimSuffix(strings.TrimSpace(prefix), "-")
	if prefix == "" {
		prefix = "id"
	}
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(buf[:]))
}
