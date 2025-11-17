package common

import (
	"encoding/json"
	"strings"
)

// MustJSON marshals the payload or panics if it fails (programming error).
func MustJSON(v any) string {
	payload, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(payload)
}

// SanitizeCLIError trims peer CLI noise to a concise message.
func SanitizeCLIError(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "unknown peer error"
	}
	if idx := strings.LastIndex(msg, "Error:"); idx != -1 {
		return strings.TrimSpace(msg[idx+6:])
	}
	return msg
}
