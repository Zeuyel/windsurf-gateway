package proxy

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

var rewritableJSONKeys = map[string]struct{}{
	"apikey":        {},
	"api_key":       {},
	"sessiontoken":  {},
	"session_token": {},
	"authtoken":     {},
	"auth_token":    {},
}

func rewriteUpstreamAuthPayload(contentType string, body []byte, backendToken string) ([]byte, error) {
	if len(body) == 0 || strings.TrimSpace(backendToken) == "" {
		return body, nil
	}

	lowerType := strings.ToLower(contentType)
	switch {
	case strings.Contains(lowerType, "application/json"), strings.Contains(lowerType, "application/connect+json"):
		return rewriteJSONAuthPayload(body, backendToken)
	case strings.Contains(lowerType, "application/connect+proto"):
		return rewriteConnectProtoAuthPayload(body, backendToken)
	case strings.Contains(lowerType, "application/proto"), strings.Contains(lowerType, "application/protobuf"), strings.Contains(lowerType, "application/connect+proto"):
		return rewriteProtoAuthPayload(body, backendToken)
	default:
		return body, nil
	}
}

func rewriteConnectProtoAuthPayload(body []byte, backendToken string) ([]byte, error) {
	if len(body) == 0 {
		return body, nil
	}

	var out bytes.Buffer
	changed := false
	idx := 0

	for idx < len(body) {
		if len(body)-idx < 5 {
			return body, fmt.Errorf("decode connect frame failed")
		}

		flags := body[idx]
		length := int(binary.BigEndian.Uint32(body[idx+1 : idx+5]))
		frameStart := idx + 5
		frameEnd := frameStart + length
		if frameEnd > len(body) {
			return body, fmt.Errorf("decode connect frame overflow")
		}

		payload := body[frameStart:frameEnd]
		replacement := payload
		if flags&0x01 == 0 {
			rewritten, err := rewriteProtoAuthPayload(payload, backendToken)
			if err != nil {
				return body, err
			}
			if !bytes.Equal(rewritten, payload) {
				replacement = rewritten
				changed = true
			}
		}

		out.WriteByte(flags)
		var lengthBytes [4]byte
		binary.BigEndian.PutUint32(lengthBytes[:], uint32(len(replacement)))
		out.Write(lengthBytes[:])
		out.Write(replacement)
		idx = frameEnd
	}

	if !changed {
		return body, nil
	}
	return out.Bytes(), nil
}

func rewriteJSONAuthPayload(body []byte, backendToken string) ([]byte, error) {
	var payload interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return body, nil
	}

	if !rewriteJSONValue(&payload, "", backendToken) {
		return body, nil
	}

	rewritten, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return rewritten, nil
}

func rewriteJSONValue(value *interface{}, parentKey, backendToken string) bool {
	switch typed := (*value).(type) {
	case map[string]interface{}:
		changed := false
		for key, nested := range typed {
			next := nested
			if rewriteJSONValue(&next, key, backendToken) {
				typed[key] = next
				changed = true
			}
		}
		return changed
	case []interface{}:
		changed := false
		for idx := range typed {
			next := typed[idx]
			if rewriteJSONValue(&next, parentKey, backendToken) {
				typed[idx] = next
				changed = true
			}
		}
		return changed
	case string:
		if shouldReplaceJSONToken(parentKey, typed) {
			*value = backendToken
			return true
		}
	}
	return false
}

func shouldReplaceJSONToken(key, value string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	if _, ok := rewritableJSONKeys[key]; ok {
		return true
	}
	return looksLikeUpstreamToken(value)
}

func rewriteProtoAuthPayload(body []byte, backendToken string) ([]byte, error) {
	rewritten, changed, err := rewriteProtoMessage(body, backendToken)
	if err != nil || !changed {
		return body, err
	}
	return rewritten, nil
}

func rewriteProtoMessage(data []byte, backendToken string) ([]byte, bool, error) {
	var out bytes.Buffer
	changed := false
	idx := 0

	for idx < len(data) {
		tagStart := idx
		tag, tagWidth, ok := decodeVarintBytes(data[idx:])
		if !ok {
			return nil, false, fmt.Errorf("decode proto tag failed")
		}
		idx += tagWidth
		fieldNumber := int(tag >> 3)
		wireType := int(tag & 0x7)

		out.Write(data[tagStart:idx])

		switch wireType {
		case 0:
			valueWidth, ok := skipVarintBytes(data[idx:])
			if !ok {
				return nil, false, fmt.Errorf("decode proto varint field %d failed", fieldNumber)
			}
			out.Write(data[idx : idx+valueWidth])
			idx += valueWidth
		case 1:
			if idx+8 > len(data) {
				return nil, false, fmt.Errorf("decode proto fixed64 field %d overflow", fieldNumber)
			}
			out.Write(data[idx : idx+8])
			idx += 8
		case 2:
			length, lengthWidth, ok := decodeVarintBytes(data[idx:])
			if !ok {
				return nil, false, fmt.Errorf("decode proto length field %d failed", fieldNumber)
			}
			idx += lengthWidth
			end := idx + int(length)
			if end > len(data) {
				return nil, false, fmt.Errorf("decode proto field %d overflow", fieldNumber)
			}

			payload := data[idx:end]
			replacement := payload
			localChanged := false

			if looksLikeTokenBytes(payload) {
				replacement = []byte(backendToken)
				localChanged = true
			}

			if !localChanged && shouldInspectNestedProto(payload) {
				nested, nestedChanged, err := rewriteProtoMessage(payload, backendToken)
				if err == nil && nestedChanged {
					replacement = nested
					localChanged = true
				}
			}

			writeVarint(&out, uint64(len(replacement)))
			out.Write(replacement)
			changed = changed || localChanged
			idx = end
		case 5:
			if idx+4 > len(data) {
				return nil, false, fmt.Errorf("decode proto fixed32 field %d overflow", fieldNumber)
			}
			out.Write(data[idx : idx+4])
			idx += 4
		default:
			return nil, false, fmt.Errorf("unsupported proto wire type %d", wireType)
		}
	}

	return out.Bytes(), changed, nil
}

func looksLikeTokenBytes(payload []byte) bool {
	return utf8.Valid(payload) && looksLikeUpstreamToken(string(payload))
}

func shouldInspectNestedProto(payload []byte) bool {
	if len(payload) == 0 {
		return false
	}

	tag := payload[0]
	fieldNumber := int(tag >> 3)
	wireType := int(tag & 0x7)
	if fieldNumber == 0 {
		return false
	}
	switch wireType {
	case 0, 1, 2, 5:
	default:
		return false
	}

	if !utf8.Valid(payload) {
		return true
	}

	for _, b := range payload {
		if b < 0x20 && b != '\t' && b != '\n' && b != '\r' {
			return true
		}
	}

	return false
}

func decodeVarintBytes(data []byte) (uint64, int, bool) {
	var value uint64
	var shift uint
	for idx, b := range data {
		value |= uint64(b&0x7f) << shift
		if b < 0x80 {
			return value, idx + 1, true
		}
		shift += 7
		if shift >= 64 {
			return 0, 0, false
		}
	}
	return 0, 0, false
}

func skipVarintBytes(data []byte) (int, bool) {
	_, width, ok := decodeVarintBytes(data)
	return width, ok
}

func writeVarint(buf *bytes.Buffer, value uint64) {
	for value >= 0x80 {
		buf.WriteByte(byte(value) | 0x80)
		value >>= 7
	}
	buf.WriteByte(byte(value))
}

func looksLikeUpstreamToken(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}

	switch {
	case strings.HasPrefix(value, "sk-ws-01-"):
		return true
	case strings.HasPrefix(value, "devin-session-token$"):
		return true
	case strings.Contains(value, "mocked-for-gateway"):
		return true
	case strings.Count(value, ".") == 2 && len(value) >= 40:
		return true
	default:
		return false
	}
}
