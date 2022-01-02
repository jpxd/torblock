package torblock_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jpxd/torblock"
)

func TestConfig(t *testing.T) {
	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusNoContent)
	})

	// Check default config works
	cfg := torblock.CreateConfig()
	_, err := torblock.New(ctx, next, cfg, "torblock")
	if err != nil {
		t.Fatalf("failed to create with default config: %s", err)
	}

	// Bad URLs have to return an error
	cfg = torblock.CreateConfig()
	cfg.AddressListURL = "bad"
	_, err = torblock.New(ctx, next, cfg, "torblock")
	if err == nil {
		t.Fatal("no error though bad address url in config")
	}

	// Unreachable URLs dont error but only warn
	cfg = torblock.CreateConfig()
	cfg.AddressListURL = "https://badurl.test123/test"
	_, err = torblock.New(ctx, next, cfg, "torblock")
	if err != nil {
		t.Fatal("unreachable url errored but should have only warned")
	}

	// Too short update intervals
	cfg = torblock.CreateConfig()
	cfg.UpdateIntervalSeconds = 1
	_, err = torblock.New(ctx, next, cfg, "torblock")
	if err == nil {
		t.Fatal("no error though to low update interval")
	}
}

func TestRequests(t *testing.T) {
	cfg := torblock.CreateConfig()
	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusNoContent)
	})

	handler, err := torblock.New(ctx, next, cfg, "torblock")
	if err != nil {
		t.Fatal(err)
	}

	// Dummy IPs
	const badIP = "176.10.99.200"
	const goodIP = "127.0.0.1"

	// Blocked IP
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RemoteAddr = fmt.Sprintf("%s:%d", badIP, 1234)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Result().StatusCode != http.StatusForbidden {
		t.Errorf("invalid status code: %d", recorder.Result().StatusCode)
	}

	// Not blocked IP
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RemoteAddr = fmt.Sprintf("%s:%d", goodIP, 1234)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Result().StatusCode != http.StatusNoContent {
		t.Errorf("invalid status code: %d", recorder.Result().StatusCode)
	}
}
