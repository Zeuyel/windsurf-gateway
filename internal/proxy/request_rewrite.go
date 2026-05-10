package proxy

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"

	"windsurf-gateway/internal/gatewayuser"
)

const (
	topLevelMetadataField = 1
	topLevelAPIKeyField   = 1

	metadataFieldAPIKey            = 3
	metadataFieldSessionID         = 10
	metadataFieldSourceAddress     = 11
	metadataFieldUserAgent         = 13
	metadataFieldExtensionPath     = 17
	metadataFieldUserID            = 20
	metadataFieldUserJWT           = 21
	metadataFieldForceTeamID       = 22
	metadataFieldDeviceFingerprint = 24
	metadataFieldTeamID            = 32
)

type upstreamAuthProfile struct {
	BackendToken            string
	StableSessionID         string
	StableUserAgent         string
	StableDeviceFingerprint string
}

var metadataRewritePaths = map[string]struct{}{
	"/exa.seat_management_pb.SeatManagementService/GetUserStatus":       {},
	"/exa.cascade_plugins_pb.CascadePluginsService/GetAllAcpRegistries": {},
	"/exa.auth_pb.AuthService/GetUserJwt":                               {},
	"/exa.api_server_pb.ApiServerService/GetStatus":                     {},
	"/exa.api_server_pb.ApiServerService/GetCliModelConfigs":            {},
	"/exa.api_server_pb.ApiServerService/GetModelStatuses":              {},
	"/exa.api_server_pb.ApiServerService/GetDefaultWorkflowTemplates":   {},
	"/exa.api_server_pb.ApiServerService/CheckChatCapacity":             {},
	"/exa.api_server_pb.ApiServerService/CheckUserMessageRateLimit":     {},
	"/exa.api_server_pb.ApiServerService/GetCommandModelConfigs":        {},
}

var apiKeyRewritePaths = map[string]struct{}{
	"/exa.seat_management_pb.SeatManagementService/MigrateApiKey":  {},
	"/exa.seat_management_pb.SeatManagementService/GetProfileData": {},
}

func rewriteUpstreamAuthPayload(requestPath, contentType string, body []byte, profile upstreamAuthProfile) ([]byte, error) {
	if len(body) == 0 || strings.TrimSpace(profile.BackendToken) == "" {
		return body, nil
	}

	lowerType := strings.ToLower(contentType)
	switch {
	case strings.Contains(lowerType, "application/json"), strings.Contains(lowerType, "application/connect+json"):
		return rewriteJSONAuthPayload(requestPath, body, profile)
	case strings.Contains(lowerType, "application/connect+proto"):
		return rewriteConnectProtoAuthPayload(requestPath, body, profile)
	case strings.Contains(lowerType, "application/proto"), strings.Contains(lowerType, "application/protobuf"):
		return rewriteProtoAuthPayload(requestPath, body, profile)
	default:
		return body, nil
	}
}

func shouldRewriteMetadata(requestPath string) bool {
	_, ok := metadataRewritePaths[requestPathWithoutQuery(requestPath)]
	return ok
}

func shouldRewriteAPIKey(requestPath string) bool {
	_, ok := apiKeyRewritePaths[requestPathWithoutQuery(requestPath)]
	return ok
}

func requestPathWithoutQuery(path string) string {
	if idx := strings.Index(path, "?"); idx >= 0 {
		return path[:idx]
	}
	return path
}

func rewriteConnectProtoAuthPayload(requestPath string, body []byte, profile upstreamAuthProfile) ([]byte, error) {
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
			rewritten, err := rewriteProtoAuthPayload(requestPath, payload, profile)
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

func rewriteProtoAuthPayload(requestPath string, body []byte, profile upstreamAuthProfile) ([]byte, error) {
	rewrittenBody := body
	var err error

	switch {
	case shouldRewriteMetadata(requestPath):
		rewrittenBody, err = rewriteTopLevelMetadataField(body, topLevelMetadataField, profile)
	case shouldRewriteAPIKey(requestPath):
		rewrittenBody, err = rewriteTopLevelStringField(body, topLevelAPIKeyField, profile.BackendToken)
	}

	if err != nil {
		return body, err
	}

	rewrittenBody, err = rewriteAnyGatewayTokenInProto(rewrittenBody, profile.BackendToken)
	if err != nil {
		return rewrittenBody, nil
	}
	return rewrittenBody, nil
}

func rewriteJSONAuthPayload(requestPath string, body []byte, profile upstreamAuthProfile) ([]byte, error) {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body, nil
	}

	changed := false
	root, ok := payload.(map[string]any)
	if !ok {
		return body, nil
	}

	switch {
	case shouldRewriteMetadata(requestPath):
		metadata, ok := root["metadata"].(map[string]any)
		if !ok {
			return body, nil
		}
		changed = rewriteJSONMetadataObject(metadata, profile)
		root["metadata"] = metadata
	case shouldRewriteAPIKey(requestPath):
		if _, ok := root["apiKey"]; ok {
			root["apiKey"] = profile.BackendToken
			changed = true
		}
		if _, ok := root["api_key"]; ok {
			root["api_key"] = profile.BackendToken
			changed = true
		}
	}

	if rewriteAnyGatewayTokenInJSON(root, profile.BackendToken) {
		changed = true
	}

	if !changed {
		return body, nil
	}

	rewritten, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}
	return rewritten, nil
}

func rewriteJSONMetadataObject(metadata map[string]any, profile upstreamAuthProfile) bool {
	changed := false

	set := func(key string, value string) {
		if current, ok := metadata[key].(string); !ok || current != value {
			metadata[key] = value
			changed = true
		}
	}

	set("apiKey", profile.BackendToken)

	return changed
}

func rewriteAnyGatewayTokenInJSON(value any, replacement string) bool {
	changed := false

	switch current := value.(type) {
	case map[string]any:
		for key, child := range current {
			switch typed := child.(type) {
			case string:
				if gatewayuser.Extract(typed) != "" && typed != replacement {
					current[key] = replacement
					changed = true
				}
			default:
				if rewriteAnyGatewayTokenInJSON(typed, replacement) {
					changed = true
				}
			}
		}
	case []any:
		for idx, child := range current {
			switch typed := child.(type) {
			case string:
				if gatewayuser.Extract(typed) != "" && typed != replacement {
					current[idx] = replacement
					changed = true
				}
			default:
				if rewriteAnyGatewayTokenInJSON(typed, replacement) {
					changed = true
				}
			}
		}
	}

	return changed
}

func rewriteAnyGatewayTokenInProto(data []byte, replacement string) ([]byte, error) {
	var out bytes.Buffer
	changed := false
	idx := 0

	for idx < len(data) {
		fieldStart := idx
		tag, tagWidth, ok := decodeVarintBytes(data[idx:])
		if !ok {
			return data, fmt.Errorf("decode proto tag failed")
		}
		idx += tagWidth

		wireType := int(tag & 0x7)
		switch wireType {
		case 0:
			valueWidth, ok := skipVarintBytes(data[idx:])
			if !ok {
				return data, fmt.Errorf("decode proto varint field failed")
			}
			out.Write(data[fieldStart : idx+valueWidth])
			idx += valueWidth
		case 1:
			if idx+8 > len(data) {
				return data, fmt.Errorf("decode proto fixed64 overflow")
			}
			out.Write(data[fieldStart : idx+8])
			idx += 8
		case 2:
			length, lengthWidth, ok := decodeVarintBytes(data[idx:])
			if !ok {
				return data, fmt.Errorf("decode proto length failed")
			}
			idx += lengthWidth
			end := idx + int(length)
			if end > len(data) {
				return data, fmt.Errorf("decode proto field overflow")
			}

			payload := data[idx:end]
			replacementPayload := payload
			localChanged := false

			if token := strings.TrimSpace(string(payload)); gatewayuser.IsToken(token) {
				replacementPayload = []byte(replacement)
				localChanged = token != replacement
			} else {
				rewrittenNested, err := rewriteAnyGatewayTokenInProto(payload, replacement)
				if err == nil && !bytes.Equal(rewrittenNested, payload) {
					replacementPayload = rewrittenNested
					localChanged = true
				}
			}

			out.Write(data[fieldStart : fieldStart+tagWidth])
			writeVarint(&out, uint64(len(replacementPayload)))
			out.Write(replacementPayload)
			changed = changed || localChanged
			idx = end
		case 5:
			if idx+4 > len(data) {
				return data, fmt.Errorf("decode proto fixed32 overflow")
			}
			out.Write(data[fieldStart : idx+4])
			idx += 4
		default:
			return data, fmt.Errorf("unsupported proto wire type %d", wireType)
		}
	}

	if !changed {
		return data, nil
	}
	return out.Bytes(), nil
}

func rewriteTopLevelMetadataField(data []byte, targetField int, profile upstreamAuthProfile) ([]byte, error) {
	rewritten, changed, err := rewriteLengthDelimitedField(data, targetField, func(payload []byte) ([]byte, bool, error) {
		return rewriteMetadataMessage(payload, profile)
	})
	if err != nil || !changed {
		return data, err
	}
	return rewritten, nil
}

func rewriteTopLevelStringField(data []byte, targetField int, replacement string) ([]byte, error) {
	rewritten, changed, err := rewriteLengthDelimitedField(data, targetField, func(payload []byte) ([]byte, bool, error) {
		if string(payload) == replacement {
			return payload, false, nil
		}
		return []byte(replacement), true, nil
	})
	if err != nil || !changed {
		return data, err
	}
	return rewritten, nil
}

func rewriteLengthDelimitedField(data []byte, targetField int, rewrite func([]byte) ([]byte, bool, error)) ([]byte, bool, error) {
	var out bytes.Buffer
	changed := false
	idx := 0

	for idx < len(data) {
		fieldStart := idx
		tag, tagWidth, ok := decodeVarintBytes(data[idx:])
		if !ok {
			return nil, false, fmt.Errorf("decode proto tag failed")
		}
		idx += tagWidth

		fieldNumber := int(tag >> 3)
		wireType := int(tag & 0x7)

		switch wireType {
		case 0:
			valueWidth, ok := skipVarintBytes(data[idx:])
			if !ok {
				return nil, false, fmt.Errorf("decode proto varint field %d failed", fieldNumber)
			}
			out.Write(data[fieldStart : idx+valueWidth])
			idx += valueWidth
		case 1:
			if idx+8 > len(data) {
				return nil, false, fmt.Errorf("decode proto fixed64 field %d overflow", fieldNumber)
			}
			out.Write(data[fieldStart : idx+8])
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
			if fieldNumber == targetField {
				replacement, localChanged, err := rewrite(payload)
				if err != nil {
					return nil, false, err
				}
				out.Write(data[fieldStart : fieldStart+tagWidth])
				writeVarint(&out, uint64(len(replacement)))
				out.Write(replacement)
				changed = changed || localChanged
			} else {
				out.Write(data[fieldStart:end])
			}
			idx = end
		case 5:
			if idx+4 > len(data) {
				return nil, false, fmt.Errorf("decode proto fixed32 field %d overflow", fieldNumber)
			}
			out.Write(data[fieldStart : idx+4])
			idx += 4
		default:
			return nil, false, fmt.Errorf("unsupported proto wire type %d", wireType)
		}
	}

	return out.Bytes(), changed, nil
}

func rewriteMetadataMessage(data []byte, profile upstreamAuthProfile) ([]byte, bool, error) {
	var out bytes.Buffer
	changed := false
	seen := map[int]bool{}
	idx := 0

	for idx < len(data) {
		fieldStart := idx
		tag, tagWidth, ok := decodeVarintBytes(data[idx:])
		if !ok {
			return nil, false, fmt.Errorf("decode metadata tag failed")
		}
		idx += tagWidth

		fieldNumber := int(tag >> 3)
		wireType := int(tag & 0x7)
		if fieldNumber == metadataFieldAPIKey {
			seen[fieldNumber] = true
		}

		switch wireType {
		case 0:
			valueWidth, ok := skipVarintBytes(data[idx:])
			if !ok {
				return nil, false, fmt.Errorf("decode metadata varint field %d failed", fieldNumber)
			}
			out.Write(data[fieldStart : idx+valueWidth])
			idx += valueWidth
		case 1:
			if idx+8 > len(data) {
				return nil, false, fmt.Errorf("decode metadata fixed64 field %d overflow", fieldNumber)
			}
			out.Write(data[fieldStart : idx+8])
			idx += 8
		case 2:
			length, lengthWidth, ok := decodeVarintBytes(data[idx:])
			if !ok {
				return nil, false, fmt.Errorf("decode metadata length field %d failed", fieldNumber)
			}
			idx += lengthWidth
			end := idx + int(length)
			if end > len(data) {
				return nil, false, fmt.Errorf("decode metadata field %d overflow", fieldNumber)
			}

			switch fieldNumber {
			case metadataFieldAPIKey:
				writeStringField(&out, fieldNumber, profile.BackendToken)
				if string(data[idx:end]) != profile.BackendToken {
					changed = true
				}
			default:
				out.Write(data[fieldStart:end])
			}
			idx = end
		case 5:
			if idx+4 > len(data) {
				return nil, false, fmt.Errorf("decode metadata fixed32 field %d overflow", fieldNumber)
			}
			out.Write(data[fieldStart : idx+4])
			idx += 4
		default:
			return nil, false, fmt.Errorf("unsupported metadata wire type %d", wireType)
		}
	}

	if !seen[metadataFieldAPIKey] {
		writeStringField(&out, metadataFieldAPIKey, profile.BackendToken)
		changed = true
	}

	return out.Bytes(), changed, nil
}

func writeStringField(buf *bytes.Buffer, fieldNumber int, value string) {
	writeVarint(buf, uint64(fieldNumber<<3|2))
	writeVarint(buf, uint64(len(value)))
	buf.WriteString(value)
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
