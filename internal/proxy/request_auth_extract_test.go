package proxy

import "testing"

func TestExtractGatewayUserTokenFromPayloadProtoMetadata(t *testing.T) {
	clientToken := "devin-session-token$abcdef1234567890"
	metadata := protoMessage(
		protoStringField(metadataFieldSessionID, "client-session"),
		protoStringField(metadataFieldAPIKey, clientToken),
		protoStringField(metadataFieldUserAgent, "connect-go/1.18.1"),
	)
	request := protoMessage(protoBytesField(topLevelMetadataField, metadata))

	got := ExtractGatewayUserTokenFromPayload("application/proto", request)
	if got != clientToken {
		t.Fatalf("ExtractGatewayUserTokenFromPayload(proto metadata) = %q, want %q", got, clientToken)
	}
}

func TestExtractGatewayUserTokenFromPayloadConnectProtoTopLevelAPIKey(t *testing.T) {
	clientToken := "devin-session-token$abcdef1234567890"
	request := protoMessage(protoStringField(topLevelAPIKeyField, clientToken))
	framed := connectProtoFrame(0, request)

	got := ExtractGatewayUserTokenFromPayload("application/connect+proto", framed)
	if got != clientToken {
		t.Fatalf("ExtractGatewayUserTokenFromPayload(connect+proto apiKey) = %q, want %q", got, clientToken)
	}
}

func TestExtractGatewayUserTokenFromPayloadJSONMetadata(t *testing.T) {
	clientToken := "devin-session-token$abcdef1234567890"
	body := []byte(`{"metadata":{"apiKey":"` + clientToken + `","sessionId":"client-session"}}`)

	got := ExtractGatewayUserTokenFromPayload("application/json", body)
	if got != clientToken {
		t.Fatalf("ExtractGatewayUserTokenFromPayload(json metadata) = %q, want %q", got, clientToken)
	}
}

func TestExtractGatewayUserTokenFromPayloadJSONTopLevelAPIKey(t *testing.T) {
	clientToken := "devin-session-token$abcdef1234567890"
	body := []byte(`{"apiKey":"` + clientToken + `"}`)

	got := ExtractGatewayUserTokenFromPayload("application/connect+json", body)
	if got != clientToken {
		t.Fatalf("ExtractGatewayUserTokenFromPayload(json apiKey) = %q, want %q", got, clientToken)
	}
}

func TestExtractGatewayUserTokenFromPayloadReturnsEmptyForPingLikeProto(t *testing.T) {
	request := protoMessage(protoStringField(2, "work"))

	got := ExtractGatewayUserTokenFromPayload("application/proto", request)
	if got != "" {
		t.Fatalf("ExtractGatewayUserTokenFromPayload(ping-like proto) = %q, want empty", got)
	}
}
