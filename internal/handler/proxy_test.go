package handler

import (
	"encoding/base64"
	"errors"
	"net/http"
	"testing"

	"windsurf-gateway/internal/proxy"
)

func TestExtractGatewayUserToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{
			name:   "gateway bearer token",
			header: "Bearer devin-session-token$abcdef1234567890",
			want:   "devin-session-token$abcdef1234567890",
		},
		{
			name:   "gateway basic token",
			header: "Basic devin-session-token$abcdef1234567890",
			want:   "devin-session-token$abcdef1234567890",
		},
		{
			name:   "gateway duplicated basic token",
			header: "Basic devin-session-token$abcdef1234567890-devin-session-token$abcdef1234567890",
			want:   "devin-session-token$abcdef1234567890",
		},
		{
			name:   "gateway standard basic username token",
			header: "Basic " + base64.StdEncoding.EncodeToString([]byte("devin-session-token$abcdef1234567890:")),
			want:   "devin-session-token$abcdef1234567890",
		},
		{
			name:   "gateway standard basic password token",
			header: "Basic " + base64.StdEncoding.EncodeToString([]byte("gateway:devin-session-token$abcdef1234567890")),
			want:   "devin-session-token$abcdef1234567890",
		},
		{
			name:   "bearer upstream token is ignored",
			header: "Bearer sk-ws-01-backend-token",
			want:   "",
		},
		{
			name:   "basic placeholder is ignored",
			header: "Basic sk-ws-01-gateway-placeholder-sk-ws-01-gateway-placeholder",
			want:   "",
		},
		{
			name:   "raw token is accepted",
			header: "devin-session-token$abcdef1234567890",
			want:   "devin-session-token$abcdef1234567890",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractGatewayUserToken(tc.header); got != tc.want {
				t.Fatalf("extractGatewayUserToken(%q) = %q, want %q", tc.header, got, tc.want)
			}
		})
	}
}

func TestClassifyProxyOutcomeOptional401DoesNotPenalizeToken(t *testing.T) {
	paths := []string{
		"/exa.api_server_pb.ApiServerService/CheckChatCapacity",
		"/exa.api_server_pb.ApiServerService/GetDefaultWorkflowTemplates",
		"/exa.auth_pb.AuthService/GetUserJwt",
		"/exa.seat_management_pb.SeatManagementService/GetProfileData",
		"/exa.seat_management_pb.SeatManagementService/MigrateApiKey",
		"/exa.cascade_plugins_pb.CascadePluginsService/GetAllAcpRegistries",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			outcome := classifyProxyOutcome(path, &proxy.ProxyResponse{
				StatusCode: http.StatusUnauthorized,
			}, nil)

			if outcome.Penalize {
				t.Fatal("expected optional 401 not to penalize token")
			}
			if outcome.FailureCategory != "optional_upstream_401" {
				t.Fatalf("unexpected failure category: %s", outcome.FailureCategory)
			}
		})
	}
}

func TestClassifyProxyOutcomeTransportErrorsDoNotCooldownToken(t *testing.T) {
	errs := []error{
		errors.New(`forward request: Post "https://server.codeium.com/exa.api_server_pb.ApiServerService/Ping": read tcp 198.18.0.1:59826->35.223.238.178:443: wsarecv: A connection attempt failed because the connected party did not properly respond after a period of time, or established connection failed because connected host has failed to respond.`),
		errors.New(`forward request: Post "https://server.codeium.com/exa.auth_pb.AuthService/GetUserJwt": context deadline exceeded`),
		errors.New(`forward request: Post "https://server.codeium.com/exa.product_analytics_pb.ProductAnalyticsService/RecordAnalyticsEvent": dial tcp: lookup server.codeium.com: no such host`),
	}

	for _, err := range errs {
		t.Run(err.Error(), func(t *testing.T) {
			outcome := classifyProxyOutcome("/exa.api_server_pb.ApiServerService/Ping", &proxy.ProxyResponse{
				StatusCode: http.StatusBadGateway,
			}, err)
			if outcome.Penalize {
				t.Fatalf("transport error should not penalize backend token: %+v", outcome)
			}
			if outcome.Cooldown != 0 {
				t.Fatalf("transport error should not set cooldown: %+v", outcome)
			}
			if outcome.Success {
				t.Fatalf("transport error should not be success: %+v", outcome)
			}
		})
	}
}

func TestBuildAnonymousAssignmentKeyUsesAuthAndIPOnly(t *testing.T) {
	keyA := buildAnonymousAssignmentKey("Basic sk-ws-01-client-abc123", "127.0.0.1")
	keyB := buildAnonymousAssignmentKey("Basic sk-ws-01-client-abc123", "127.0.0.2")
	keyC := buildAnonymousAssignmentKey("Basic sk-ws-01-client-other", "127.0.0.1")

	if keyA != keyB {
		t.Fatalf("expected per-client placeholder auth to be stable across IPs: %s vs %s", keyA, keyB)
	}
	if keyA == keyC {
		t.Fatalf("expected different placeholder auth to produce different keys: %s vs %s", keyA, keyC)
	}
}

func TestBuildAnonymousAssignmentKeyFallsBackToIPForLegacyPlaceholder(t *testing.T) {
	keyA := buildAnonymousAssignmentKey("Basic sk-ws-01-gateway-placeholder", "127.0.0.1")
	keyB := buildAnonymousAssignmentKey("Basic sk-ws-01-gateway-placeholder", "127.0.0.2")

	if keyA == keyB {
		t.Fatalf("expected legacy shared placeholder auth to keep IP in the key: %s vs %s", keyA, keyB)
	}
}

func TestShouldRequireWindsurfQuotaForPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "/exa.seat_management_pb.SeatManagementService/GetUserStatus", want: false},
		{path: "/exa.api_server_pb.ApiServerService/Ping", want: false},
		{path: "/exa.api_server_pb.ApiServerService/GetCliModelConfigs", want: false},
		{path: "/exa.product_analytics_pb.ProductAnalyticsService/RecordAnalyticsEvent", want: false},
		{path: "/exa.analytics_pb.AnalyticsService/RecordCortexTrajectoryStep", want: false},
		{path: "/exa.api_server_pb.ApiServerService/RecordAsyncTelemetry", want: false},
		{path: "/exa.api_server_pb.ApiServerService/CheckChatCapacity", want: false},
		{path: "/exa.api_server_pb.ApiServerService/CheckUserMessageRateLimit", want: false},
		{path: "/exa.api_server_pb.ApiServerService/GetChatMessage", want: true},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			if got := shouldRequireWindsurfQuotaForPath(tc.path); got != tc.want {
				t.Fatalf("shouldRequireWindsurfQuotaForPath(%q) = %t, want %t", tc.path, got, tc.want)
			}
		})
	}
}

func TestShouldRequireGatewayUserAuthForPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "/exa.api_server_pb.ApiServerService/Ping", want: false},
		{path: "/exa.api_server_pb.ApiServerService/GetStatus", want: false},
		{path: "/exa.api_server_pb.ApiServerService/CheckChatCapacity", want: false},
		{path: "/exa.product_analytics_pb.ProductAnalyticsService/RecordAnalyticsEvent", want: false},
		{path: "/exa.api_server_pb.ApiServerService/GetChatMessage", want: true},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			if got := shouldRequireGatewayUserAuthForPath(tc.path); got != tc.want {
				t.Fatalf("shouldRequireGatewayUserAuthForPath(%q) = %t, want %t", tc.path, got, tc.want)
			}
		})
	}
}

func TestIsTelemetryEndpoint(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "/exa.product_analytics_pb.ProductAnalyticsService/RecordAnalyticsEvent", want: true},
		{path: "/exa.product_analytics_pb.ProductAnalyticsService/BatchRecordAnalyticsEvents", want: true},
		{path: "/exa.analytics_pb.AnalyticsService/RecordCortexTrajectoryStep", want: true},
		{path: "/exa.api_server_pb.ApiServerService/RecordAsyncTelemetry", want: true},
		{path: "/exa.api_server_pb.ApiServerService/RecordCommitMessageSave", want: true},
		{path: "/exa.language_server_pb.LanguageServerService/RecordUserGrep", want: true},
		{path: "/exa.seat_management_pb.SeatManagementService/UpdateCodeSnippetTelemetry", want: true},
		{path: "/exa.api_server_pb.ApiServerService/GetChatMessage", want: false},
		{path: "/exa.seat_management_pb.SeatManagementService/GetUserStatus", want: false},
		{path: "/exa.api_server_pb.ApiServerService/RecordChatPanelSession", want: false},
		{path: "/exa.api_server_pb.ApiServerService/BatchRecordChatRequestRecords", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			if got := isTelemetryEndpoint(tc.path); got != tc.want {
				t.Fatalf("isTelemetryEndpoint(%q) = %t, want %t", tc.path, got, tc.want)
			}
		})
	}
}
