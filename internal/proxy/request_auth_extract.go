package proxy

import (
	"encoding/binary"
	"encoding/json"
	"strings"

	"windsurf-gateway/internal/gatewayuser"
)

// ExtractGatewayUserTokenFromPayload reads Windsurf client auth from request
// bodies. Windsurf commonly sends the gateway token in protobuf metadata.apiKey
// or top-level apiKey fields instead of HTTP auth headers.
func ExtractGatewayUserTokenFromPayload(contentType string, body []byte) string {
	if len(body) == 0 {
		return ""
	}

	lowerType := strings.ToLower(strings.TrimSpace(contentType))
	switch {
	case strings.Contains(lowerType, "application/json"), strings.Contains(lowerType, "application/connect+json"):
		if token := extractGatewayUserTokenFromJSON(body); token != "" {
			return token
		}
	case strings.Contains(lowerType, "application/connect+proto"):
		if token := extractGatewayUserTokenFromConnectProto(body); token != "" {
			return token
		}
	case strings.Contains(lowerType, "application/proto"),
		strings.Contains(lowerType, "application/protobuf"),
		strings.Contains(lowerType, "application/octet-stream"):
		if token := extractGatewayUserTokenFromProto(body); token != "" {
			return token
		}
	}

	if token := extractGatewayUserTokenFromJSON(body); token != "" {
		return token
	}
	if token := extractGatewayUserTokenFromConnectProto(body); token != "" {
		return token
	}
	return extractGatewayUserTokenFromProto(body)
}

func extractGatewayUserTokenFromJSON(body []byte) string {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}

	root, ok := payload.(map[string]any)
	if !ok {
		return ""
	}

	if token := extractGatewayUserTokenFromJSONObject(root); token != "" {
		return token
	}

	metadata, ok := root["metadata"].(map[string]any)
	if !ok {
		return ""
	}
	return extractGatewayUserTokenFromJSONObject(metadata)
}

func extractGatewayUserTokenFromJSONObject(root map[string]any) string {
	for _, key := range []string{"apiKey", "api_key", "token"} {
		value, ok := root[key].(string)
		if !ok {
			continue
		}
		if token := gatewayuser.Extract(value); token != "" {
			return token
		}
	}
	return ""
}

func extractGatewayUserTokenFromConnectProto(body []byte) string {
	idx := 0
	for idx < len(body) {
		if len(body)-idx < 5 {
			return ""
		}

		flags := body[idx]
		length := int(binary.BigEndian.Uint32(body[idx+1 : idx+5]))
		frameStart := idx + 5
		frameEnd := frameStart + length
		if length < 0 || frameEnd > len(body) {
			return ""
		}

		if flags&0x01 == 0 {
			if token := extractGatewayUserTokenFromProto(body[frameStart:frameEnd]); token != "" {
				return token
			}
		}

		idx = frameEnd
	}
	return ""
}

func extractGatewayUserTokenFromProto(body []byte) string {
	idx := 0
	for idx < len(body) {
		tag, tagWidth, ok := decodeVarintBytes(body[idx:])
		if !ok {
			return ""
		}
		idx += tagWidth

		fieldNumber := int(tag >> 3)
		wireType := int(tag & 0x7)

		switch wireType {
		case 0:
			valueWidth, ok := skipVarintBytes(body[idx:])
			if !ok {
				return ""
			}
			idx += valueWidth
		case 1:
			if idx+8 > len(body) {
				return ""
			}
			idx += 8
		case 2:
			length, lengthWidth, ok := decodeVarintBytes(body[idx:])
			if !ok {
				return ""
			}
			idx += lengthWidth
			end := idx + int(length)
			if end > len(body) {
				return ""
			}

			payload := body[idx:end]
			if fieldNumber == topLevelMetadataField {
				if token := gatewayuser.Extract(string(payload)); token != "" {
					return token
				}
				if token := extractGatewayUserTokenFromMetadataProto(payload); token != "" {
					return token
				}
			}
			idx = end
		case 5:
			if idx+4 > len(body) {
				return ""
			}
			idx += 4
		default:
			return ""
		}
	}
	return ""
}

func extractGatewayUserTokenFromMetadataProto(body []byte) string {
	idx := 0
	for idx < len(body) {
		tag, tagWidth, ok := decodeVarintBytes(body[idx:])
		if !ok {
			return ""
		}
		idx += tagWidth

		fieldNumber := int(tag >> 3)
		wireType := int(tag & 0x7)

		switch wireType {
		case 0:
			valueWidth, ok := skipVarintBytes(body[idx:])
			if !ok {
				return ""
			}
			idx += valueWidth
		case 1:
			if idx+8 > len(body) {
				return ""
			}
			idx += 8
		case 2:
			length, lengthWidth, ok := decodeVarintBytes(body[idx:])
			if !ok {
				return ""
			}
			idx += lengthWidth
			end := idx + int(length)
			if end > len(body) {
				return ""
			}

			if fieldNumber == metadataFieldAPIKey {
				if token := gatewayuser.Extract(string(body[idx:end])); token != "" {
					return token
				}
			}
			idx = end
		case 5:
			if idx+4 > len(body) {
				return ""
			}
			idx += 4
		default:
			return ""
		}
	}
	return ""
}
