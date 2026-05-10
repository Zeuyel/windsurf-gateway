package proxy

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestRewriteProtoAuthPayloadRewritesMetadataAndPrivacyFields(t *testing.T) {
	profile := upstreamAuthProfile{
		BackendToken:            "sk-ws-01-backend",
		StableSessionID:         "stable-session",
		StableUserAgent:         "WindsurfGateway/test",
		StableDeviceFingerprint: "stable-device",
	}
	clientToken := "devin-session-token$abcdef1234567890"

	metadata := protoMessage(
		protoStringField(metadataFieldAPIKey, clientToken),
		protoStringField(metadataFieldSessionID, "client-session"),
		protoStringField(metadataFieldSourceAddress, "127.0.0.1"),
		protoStringField(metadataFieldUserAgent, "connect-go/1.0"),
		protoStringField(metadataFieldExtensionPath, "/home/user/.windsurf"),
		protoStringField(metadataFieldUserID, "user-123"),
		protoStringField(metadataFieldUserJWT, "jwt.jwt.jwt"),
		protoStringField(metadataFieldForceTeamID, "team-force"),
		protoStringField(metadataFieldDeviceFingerprint, "client-device"),
		protoStringField(metadataFieldTeamID, "team-1"),
	)
	request := protoMessage(protoBytesField(1, metadata))

	rewritten, err := rewriteProtoAuthPayload("/exa.seat_management_pb.SeatManagementService/GetUserStatus", request, profile)
	if err != nil {
		t.Fatalf("rewriteProtoAuthPayload returned error: %v", err)
	}

	if bytes.Equal(rewritten, request) {
		t.Fatal("expected protobuf payload to be rewritten")
	}
	for _, forbidden := range []string{
		clientToken,
		"127.0.0.1",
		"/home/user/.windsurf",
		"user-123",
		"jwt.jwt.jwt",
		"team-force",
		"team-1",
		"client-device",
	} {
		if bytes.Contains(rewritten, []byte(forbidden)) {
			t.Fatalf("unexpected client value still present after rewrite: %s", forbidden)
		}
	}
	for _, expected := range []string{
		profile.BackendToken,
		profile.StableSessionID,
		profile.StableUserAgent,
		profile.StableDeviceFingerprint,
	} {
		if !bytes.Contains(rewritten, []byte(expected)) {
			t.Fatalf("expected rewritten payload to contain %s", expected)
		}
	}
}

func TestRewriteUpstreamAuthPayloadIgnoresUnknownBinaryFieldsOnKnownPath(t *testing.T) {
	profile := upstreamAuthProfile{
		BackendToken:            "sk-ws-01-backend",
		StableSessionID:         "stable-session",
		StableUserAgent:         "WindsurfGateway/test",
		StableDeviceFingerprint: "stable-device",
	}

	metadata := protoMessage(
		protoStringField(metadataFieldAPIKey, "devin-session-token$abcdef1234567890"),
		protoStringField(metadataFieldUserAgent, "client-agent"),
	)
	request := protoMessage(
		protoBytesField(1, metadata),
		protoBytesField(2, []byte{0x07, 0x01, 0x02, 0x03}),
	)
	framed := connectProtoFrame(0, request)

	rewritten, err := rewriteUpstreamAuthPayload("/exa.api_server_pb.ApiServerService/GetDefaultWorkflowTemplates", "application/connect+proto", framed, profile)
	if err != nil {
		t.Fatalf("rewriteUpstreamAuthPayload returned error: %v", err)
	}
	if bytes.Equal(rewritten, framed) {
		t.Fatal("expected connect proto payload to be rewritten")
	}
	if binary.BigEndian.Uint32(rewritten[1:5]) == 0 {
		t.Fatal("expected non-empty connect frame")
	}
	if bytes.Contains(rewritten, []byte("client-agent")) {
		t.Fatal("client user agent should have been rewritten")
	}
	if !bytes.Contains(rewritten, []byte(profile.StableUserAgent)) {
		t.Fatal("stable upstream user agent missing after rewrite")
	}
}

func TestRewriteUpstreamAuthPayloadRewritesTopLevelAPIKeyJSON(t *testing.T) {
	profile := upstreamAuthProfile{BackendToken: "sk-ws-01-backend"}
	payload := []byte(`{"apiKey":"devin-session-token$abcdef1234567890"}`)

	rewritten, err := rewriteUpstreamAuthPayload("/exa.seat_management_pb.SeatManagementService/MigrateApiKey", "application/json", payload, profile)
	if err != nil {
		t.Fatalf("rewriteUpstreamAuthPayload returned error: %v", err)
	}
	if bytes.Equal(rewritten, payload) {
		t.Fatal("expected JSON payload to be rewritten")
	}
	if bytes.Contains(rewritten, []byte("devin-session-token$abcdef1234567890")) {
		t.Fatal("gateway user token still present after rewrite")
	}
	if !bytes.Contains(rewritten, []byte(profile.BackendToken)) {
		t.Fatal("backend token missing after JSON rewrite")
	}
}

func TestRewriteUpstreamAuthPayloadRewritesGatewayTokenOnUnknownProtoPath(t *testing.T) {
	profile := upstreamAuthProfile{BackendToken: "sk-ws-01-backend"}
	request := protoMessage(protoStringField(7, "devin-session-token$abcdef1234567890"))

	rewritten, err := rewriteUpstreamAuthPayload("/exa.auth_pb.AuthService/GetUserJwt", "application/proto", request, profile)
	if err != nil {
		t.Fatalf("rewriteUpstreamAuthPayload returned error: %v", err)
	}
	if bytes.Equal(rewritten, request) {
		t.Fatal("expected proto payload to be rewritten")
	}
	if bytes.Contains(rewritten, []byte("devin-session-token$abcdef1234567890")) {
		t.Fatal("gateway user token still present after generic proto rewrite")
	}
	if !bytes.Contains(rewritten, []byte(profile.BackendToken)) {
		t.Fatal("backend token missing after generic proto rewrite")
	}
}

func TestRewriteUpstreamAuthPayloadRewritesGetUserJwtMetadataProto(t *testing.T) {
	profile := upstreamAuthProfile{
		BackendToken:            "sk-ws-01-backend",
		StableSessionID:         "stable-session",
		StableUserAgent:         "WindsurfGateway/test",
		StableDeviceFingerprint: "stable-device",
	}
	request := protoMessage(protoBytesField(topLevelMetadataField, protoMessage(
		protoStringField(metadataFieldAPIKey, "devin-session-token$abcdef1234567890"),
		protoStringField(metadataFieldSessionID, "client-session"),
	)))

	rewritten, err := rewriteUpstreamAuthPayload("/exa.auth_pb.AuthService/GetUserJwt", "application/proto", request, profile)
	if err != nil {
		t.Fatalf("rewriteUpstreamAuthPayload returned error: %v", err)
	}
	if bytes.Equal(rewritten, request) {
		t.Fatal("expected GetUserJwt proto payload to be rewritten")
	}
	if bytes.Contains(rewritten, []byte("devin-session-token$abcdef1234567890")) {
		t.Fatal("gateway user token still present after GetUserJwt metadata rewrite")
	}
	for _, expected := range []string{profile.BackendToken, profile.StableSessionID} {
		if !bytes.Contains(rewritten, []byte(expected)) {
			t.Fatalf("expected rewritten payload to contain %s", expected)
		}
	}
}

func protoMessage(fields ...[]byte) []byte {
	var out bytes.Buffer
	for _, field := range fields {
		out.Write(field)
	}
	return out.Bytes()
}

func protoStringField(fieldNumber int, value string) []byte {
	return protoBytesField(fieldNumber, []byte(value))
}

func protoBytesField(fieldNumber int, value []byte) []byte {
	var out bytes.Buffer
	writeVarint(&out, uint64(fieldNumber<<3|2))
	writeVarint(&out, uint64(len(value)))
	out.Write(value)
	return out.Bytes()
}

func connectProtoFrame(flags byte, payload []byte) []byte {
	var out bytes.Buffer
	out.WriteByte(flags)
	var lengthBytes [4]byte
	binary.BigEndian.PutUint32(lengthBytes[:], uint32(len(payload)))
	out.Write(lengthBytes[:])
	out.Write(payload)
	return out.Bytes()
}
