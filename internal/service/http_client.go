package service

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"
)

func newExternalHTTPClient(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		MaxConnsPerHost:       100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
		ForceAttemptHTTP2:     false,
		TLSNextProto:          make(map[string]func(string, *tls.Conn) http.RoundTripper),
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

func doRequestWithRetry(client *http.Client, attempts int, build func() (*http.Request, error)) (*http.Response, error) {
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		req, err := build()
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(req)
		if err == nil {
			return resp, nil
		}

		lastErr = err
		if !isRetryableTransportError(err) || attempt == attempts {
			return nil, err
		}

		client.CloseIdleConnections()
		time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
	}

	return nil, lastErr
}

func isRetryableTransportError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
		return true
	}

	lower := strings.ToLower(err.Error())
	retryableSubstrings := []string{
		"unexpected eof",
		"server closed idle connection",
		"connection reset by peer",
		"broken pipe",
		"use of closed network connection",
		"http2: client connection lost",
		"stream error",
		"goaway",
	}
	for _, substring := range retryableSubstrings {
		if strings.Contains(lower, substring) {
			return true
		}
	}

	return false
}
