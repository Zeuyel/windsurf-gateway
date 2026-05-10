package service

import "testing"

func TestQuotaSyncPassiveMessage(t *testing.T) {
	if QuotaSyncPassiveMessage() == "" {
		t.Fatal("expected passive quota sync message")
	}
}
