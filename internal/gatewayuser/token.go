package gatewayuser

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

const (
	TokenPrefix      = "devin-session-token$"
	tokenHexByteSize = 16
)

func Generate() (string, error) {
	buf := make([]byte, tokenHexByteSize)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return TokenPrefix + hex.EncodeToString(buf), nil
}

func IsToken(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) <= len(TokenPrefix) || !strings.HasPrefix(strings.ToLower(value), TokenPrefix) {
		return false
	}
	for _, ch := range value[len(TokenPrefix):] {
		switch {
		case ch >= '0' && ch <= '9':
		case ch >= 'a' && ch <= 'f':
		case ch >= 'A' && ch <= 'F':
		default:
			return false
		}
	}
	return true
}

func Extract(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	lower := strings.ToLower(value)
	switch {
	case strings.HasPrefix(lower, "bearer "):
		return extractRawToken(strings.TrimSpace(value[7:]))
	case strings.HasPrefix(lower, "basic "):
		value = strings.TrimSpace(value[6:])
		if token := extractRawToken(value); token != "" {
			return token
		}
		if decoded := decodeBasicValue(value); decoded != "" {
			return extractRawToken(decoded)
		}
		return ""
	default:
		return extractRawToken(value)
	}
}

func extractRawToken(value string) string {
	lower := strings.ToLower(value)
	idx := strings.Index(lower, TokenPrefix)
	if idx < 0 {
		return ""
	}

	token := value[idx : idx+len(TokenPrefix)]
	for _, ch := range value[idx+len(TokenPrefix):] {
		switch {
		case ch >= '0' && ch <= '9':
			token += string(ch)
		case ch >= 'a' && ch <= 'f':
			token += string(ch)
		case ch >= 'A' && ch <= 'F':
			token += string(ch)
		default:
			if IsToken(token) {
				return token
			}
			return ""
		}
	}

	if IsToken(token) {
		return token
	}
	return ""
}

func decodeBasicValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, encoding := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	} {
		decoded, err := encoding.DecodeString(value)
		if err == nil {
			return strings.TrimSpace(string(decoded))
		}
	}
	return ""
}
