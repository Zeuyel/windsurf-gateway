package proxy

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestRewriteProtoAuthPayloadSanitizesMetadata(t *testing.T) {
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
	if bytes.Contains(rewritten, []byte(clientToken)) {
		t.Fatal("gateway user token still present after rewrite")
	}
	for _, removed := range []string{
		"client-session",
		"127.0.0.1",
		"/home/user/.windsurf",
		"user-123",
		"jwt.jwt.jwt",
		"team-force",
		"client-device",
		"team-1",
	} {
		if bytes.Contains(rewritten, []byte(removed)) {
			t.Fatalf("expected private metadata to be removed or replaced: %s", removed)
		}
	}
	if !bytes.Contains(rewritten, []byte("connect-go/1.0")) {
		t.Fatal("client user agent metadata should be preserved")
	}
	if !bytes.Contains(rewritten, []byte(profile.BackendToken)) {
		t.Fatal("backend token missing after metadata rewrite")
	}
	if !bytes.Contains(rewritten, []byte(profile.StableSessionID)) {
		t.Fatal("stable session id missing after metadata rewrite")
	}
	if !bytes.Contains(rewritten, []byte(profile.StableDeviceFingerprint)) {
		t.Fatal("stable device fingerprint missing after metadata rewrite")
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
	if !bytes.Contains(rewritten, []byte("client-agent")) {
		t.Fatal("client user agent metadata should be preserved")
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

func TestRewriteUpstreamAuthPayloadRewritesGetProfileDataAPIKeyProto(t *testing.T) {
	profile := upstreamAuthProfile{BackendToken: "sk-ws-01-backend"}
	request := protoMessage(protoStringField(topLevelAPIKeyField, "devin-session-token$abcdef1234567890"))

	rewritten, err := rewriteUpstreamAuthPayload("/exa.seat_management_pb.SeatManagementService/GetProfileData", "application/proto", request, profile)
	if err != nil {
		t.Fatalf("rewriteUpstreamAuthPayload returned error: %v", err)
	}
	if bytes.Equal(rewritten, request) {
		t.Fatal("expected GetProfileData api_key to be rewritten")
	}
	if bytes.Contains(rewritten, []byte("devin-session-token$abcdef1234567890")) {
		t.Fatal("gateway user token still present after GetProfileData rewrite")
	}
	if !bytes.Contains(rewritten, []byte(profile.BackendToken)) {
		t.Fatal("backend token missing after GetProfileData rewrite")
	}
}

func TestRewriteProtoAuthPayloadKeepsMetadataRewriteWhenGenericScanFails(t *testing.T) {
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
		protoBytesField(topLevelMetadataField, metadata),
		protoBytesField(2, []byte{0x0f}),
	)

	rewritten, err := rewriteProtoAuthPayload("/exa.api_server_pb.ApiServerService/CheckChatCapacity", request, profile)
	if err != nil {
		t.Fatalf("rewriteProtoAuthPayload returned error: %v", err)
	}
	if bytes.Contains(rewritten, []byte("devin-session-token$abcdef1234567890")) {
		t.Fatal("gateway user token still present in metadata after rewrite")
	}
	if !bytes.Contains(rewritten, []byte(profile.BackendToken)) {
		t.Fatal("backend token missing after metadata rewrite")
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
	if !bytes.Contains(rewritten, []byte(profile.BackendToken)) {
		t.Fatalf("expected rewritten payload to contain %s", profile.BackendToken)
	}
	if bytes.Contains(rewritten, []byte("client-session")) {
		t.Fatal("client session metadata should be replaced")
	}
	if !bytes.Contains(rewritten, []byte(profile.StableSessionID)) {
		t.Fatal("stable session metadata should be present")
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
