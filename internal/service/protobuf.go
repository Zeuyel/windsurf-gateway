package service

import (
	"fmt"
	"strings"
)

func encodeProtoStringField(fieldNumber int, value string) []byte {
	buf := make([]byte, 0, len(value)+4)
	appendProtoStringField(&buf, fieldNumber, value)
	return buf
}

func appendProtoStringField(buf *[]byte, fieldNumber int, value string) {
	appendVarint(buf, uint64(fieldNumber<<3|2))
	appendVarint(buf, uint64(len(value)))
	*buf = append(*buf, value...)
}

func appendVarint(buf *[]byte, value uint64) {
	for value >= 0x80 {
		*buf = append(*buf, byte(value)|0x80)
		value >>= 7
	}
	*buf = append(*buf, byte(value))
}

func decodeVarint(data []byte, start int) (uint64, int, bool) {
	var value uint64
	var shift uint
	for i := start; i < len(data); i++ {
		b := data[i]
		value |= uint64(b&0x7f) << shift
		if b < 0x80 {
			return value, i - start + 1, true
		}
		shift += 7
		if shift >= 64 {
			return 0, 0, false
		}
	}
	return 0, 0, false
}

func parseCheckUserLoginMethodResponse(data []byte) (*CheckUserLoginMethodResult, error) {
	result := &CheckUserLoginMethodResult{}
	idx := 0
	for idx < len(data) {
		tag, consumed, ok := decodeVarint(data, idx)
		if !ok {
			return nil, fmt.Errorf("decode CheckUserLoginMethod tag failed")
		}
		idx += consumed

		fieldNumber := int(tag >> 3)
		wireType := int(tag & 0x7)

		switch wireType {
		case 0:
			value, width, ok := decodeVarint(data, idx)
			if !ok {
				return nil, fmt.Errorf("decode CheckUserLoginMethod field %d failed", fieldNumber)
			}
			switch fieldNumber {
			case 2:
				result.DisallowEnterpriseUserLogin = value != 0
			case 3:
				result.UserExists = value != 0
			case 4:
				result.IsMigrated = value != 0
			case 5:
				result.HasPassword = value != 0
			}
			idx += width
		case 2:
			length, width, ok := decodeVarint(data, idx)
			if !ok {
				return nil, fmt.Errorf("decode CheckUserLoginMethod field %d length failed", fieldNumber)
			}
			idx += width
			end := idx + int(length)
			if end > len(data) {
				return nil, fmt.Errorf("CheckUserLoginMethod field %d exceeds payload", fieldNumber)
			}
			if fieldNumber == 1 {
				result.RedirectURL = string(data[idx:end])
			}
			idx = end
		case 1:
			idx += 8
		case 5:
			idx += 4
		default:
			return nil, fmt.Errorf("unsupported CheckUserLoginMethod wire type %d", wireType)
		}
		if idx > len(data) {
			return nil, fmt.Errorf("CheckUserLoginMethod parse overflow")
		}
	}
	return result, nil
}

func parseLengthDelimitedStringField(data []byte, targetField int) (string, error) {
	idx := 0
	for idx < len(data) {
		tag, consumed, ok := decodeVarint(data, idx)
		if !ok {
			return "", fmt.Errorf("decode tag failed")
		}
		idx += consumed

		fieldNumber := int(tag >> 3)
		wireType := int(tag & 0x7)
		switch wireType {
		case 0:
			_, width, ok := decodeVarint(data, idx)
			if !ok {
				return "", fmt.Errorf("decode field %d failed", fieldNumber)
			}
			idx += width
		case 2:
			length, width, ok := decodeVarint(data, idx)
			if !ok {
				return "", fmt.Errorf("decode field %d length failed", fieldNumber)
			}
			idx += width
			end := idx + int(length)
			if end > len(data) {
				return "", fmt.Errorf("field %d exceeds payload", fieldNumber)
			}
			if fieldNumber == targetField {
				return string(data[idx:end]), nil
			}
			idx = end
		case 1:
			idx += 8
		case 5:
			idx += 4
		default:
			return "", fmt.Errorf("unsupported wire type %d", wireType)
		}
		if idx > len(data) {
			return "", fmt.Errorf("parse overflow")
		}
	}
	return "", nil
}

func boolPtr(value bool) *bool {
	v := value
	return &v
}

func defaultBrowserHeaders(origin, referer string) map[string]string {
	return map[string]string{
		"Accept":          "*/*",
		"Accept-Language": "zh-CN,zh;q=0.9",
		"Origin":          origin,
		"Referer":         referer,
		"User-Agent":      defaultBrowserUserAgent,
		"Sec-Fetch-Dest":  "empty",
		"Sec-Fetch-Mode":  "cors",
		"Sec-Fetch-Site":  "same-origin",
	}
}

func applyHeaders(reqHeaderSetter interface{ Set(string, string) }, headers map[string]string) {
	for key, value := range headers {
		if strings.TrimSpace(value) == "" {
			continue
		}
		reqHeaderSetter.Set(key, value)
	}
}
