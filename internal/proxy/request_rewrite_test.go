package proxy

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestRewriteProtoAuthPayloadRewritesNestedValidUTF8Message(t *testing.T) {
	backendToken := "sk-ws-01-backend"
	clientToken := "sk-ws-01-client"

	metadata := protoMessage(
		protoStringField(1, "windsurf"),
		protoStringField(2, "1.0.0"),
		protoStringField(6, clientToken),
	)
	request := protoMessage(protoBytesField(1, metadata))

	rewritten, err := rewriteProtoAuthPayload(request, backendToken)
	if err != nil {
		t.Fatalf("rewriteProtoAuthPayload returned error: %v", err)
	}

	if bytes.Equal(rewritten, request) {
		t.Fatal("expected protobuf payload to be rewritten")
	}
	if bytes.Contains(rewritten, []byte(clientToken)) {
		t.Fatal("client token still present after rewrite")
	}
	if !bytes.Contains(rewritten, []byte(backendToken)) {
		t.Fatal("backend token missing after rewrite")
	}
}

func TestRewriteUpstreamAuthPayloadRewritesConnectProtoEnvelope(t *testing.T) {
	backendToken := "sk-ws-01-backend"
	clientToken := "sk-ws-01-client"

	message := protoMessage(protoStringField(1, clientToken))
	framed := connectProtoFrame(0, message)

	rewritten, err := rewriteUpstreamAuthPayload("application/connect+proto", framed, backendToken)
	if err != nil {
		t.Fatalf("rewriteUpstreamAuthPayload returned error: %v", err)
	}

	if bytes.Equal(rewritten, framed) {
		t.Fatal("expected connect proto payload to be rewritten")
	}
	if binary.BigEndian.Uint32(rewritten[1:5]) != uint32(len(message)-len(clientToken)+len(backendToken)) {
		t.Fatal("connect frame length was not updated")
	}
	if bytes.Contains(rewritten, []byte(clientToken)) {
		t.Fatal("client token still present after connect rewrite")
	}
	if !bytes.Contains(rewritten, []byte(backendToken)) {
		t.Fatal("backend token missing after connect rewrite")
	}
}

func TestRewriteUpstreamAuthPayloadRewritesGatewayUserToken(t *testing.T) {
	backendToken := "sk-ws-01-backend"
	clientToken := "ws-abc123"

	message := protoMessage(protoStringField(1, clientToken))
	framed := connectProtoFrame(0, message)

	rewritten, err := rewriteUpstreamAuthPayload("application/connect+proto", framed, backendToken)
	if err != nil {
		t.Fatalf("rewriteUpstreamAuthPayload returned error: %v", err)
	}
	if bytes.Equal(rewritten, framed) {
		t.Fatal("expected gateway user token payload to be rewritten")
	}
	if bytes.Contains(rewritten, []byte(clientToken)) {
		t.Fatal("gateway user token still present after rewrite")
	}
	if !bytes.Contains(rewritten, []byte(backendToken)) {
		t.Fatal("backend token missing after rewrite")
	}
}

func TestRewriteJSONAuthPayloadRewritesBasicGatewayCredential(t *testing.T) {
	backendToken := "sk-ws-01-backend"
	payload := []byte(`{"apiKey":"Basic ws-abc123-ws-abc123"}`)

	rewritten, err := rewriteUpstreamAuthPayload("application/json", payload, backendToken)
	if err != nil {
		t.Fatalf("rewriteUpstreamAuthPayload returned error: %v", err)
	}
	if bytes.Equal(rewritten, payload) {
		t.Fatal("expected basic gateway credential JSON payload to be rewritten")
	}
	if bytes.Contains(rewritten, []byte("ws-abc123")) {
		t.Fatal("gateway user credential still present after rewrite")
	}
	if !bytes.Contains(rewritten, []byte(backendToken)) {
		t.Fatal("backend token missing after JSON rewrite")
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
