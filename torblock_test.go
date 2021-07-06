package torblock_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jpxd/torblock"
)

func TestMain(m *testing.M) {
	// Disable logging output in tests
	log.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

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

	// Too short update intervals
	cfg = torblock.CreateConfig()
	cfg.UpdateInterval = 1
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

	// Blocked IP in RemoteAddr
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

	// Blocked IP in X-Forwarded-For
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("X-Forwarded-For", fmt.Sprintf("%s, %s", goodIP, badIP))
	recorder = httptest.NewRecorder()
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
