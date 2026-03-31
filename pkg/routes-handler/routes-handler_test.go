package routeshandler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestExpose_FirstURLSucceeds(t *testing.T) {
	// atomic.Int32 makes the cross-goroutine access explicit: httptest.Server
	// handles each request in its own goroutine, so the handler increments
	// happen in a different goroutine than the test assertions. In practice
	// the HTTP round-trip synchronisation is enough, but atomic keeps it
	// self-documenting and safe regardless.
	var calledFirstURL, calledSecondURL atomic.Int32

	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledFirstURL.Add(1)
		assertHostPayload(t, r, "app.example.com")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv1.Close()

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calledSecondURL.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv2.Close()

	overrideURLs(t, &addHostURLs, []string{srv1.URL, srv2.URL})

	if err := expose("app.example.com"); err != nil {
		t.Fatalf("expose returned unexpected error: %v", err)
	}
	if calledFirstURL.Load() != 1 {
		t.Errorf("expected first URL to be called once, got %d", calledFirstURL.Load())
	}
	if calledSecondURL.Load() != 0 {
		t.Errorf("expected second URL not to be called, got %d", calledSecondURL.Load())
	}
}

func TestExpose_FirstURLFails_SecondSucceeds(t *testing.T) {
	var calledFirstURL, calledSecondURL atomic.Int32

	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calledFirstURL.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv1.Close()

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledSecondURL.Add(1)
		assertHostPayload(t, r, "app.example.com")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv2.Close()

	overrideURLs(t, &addHostURLs, []string{srv1.URL, srv2.URL})

	if err := expose("app.example.com"); err != nil {
		t.Fatalf("expose returned unexpected error: %v", err)
	}
	if calledFirstURL.Load() != 1 {
		t.Errorf("expected first URL to be called once, got %d", calledFirstURL.Load())
	}
	if calledSecondURL.Load() != 1 {
		t.Errorf("expected second URL to be called once, got %d", calledSecondURL.Load())
	}
}

func TestUnexpose_FirstURLSucceeds(t *testing.T) {
	var calledFirstURL, calledSecondURL atomic.Int32

	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledFirstURL.Add(1)
		assertHostPayload(t, r, "old.example.com")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv1.Close()

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calledSecondURL.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv2.Close()

	overrideURLs(t, &removeHostURLs, []string{srv1.URL, srv2.URL})

	if err := unexpose("old.example.com"); err != nil {
		t.Fatalf("unexpose returned unexpected error: %v", err)
	}
	if calledFirstURL.Load() != 1 {
		t.Errorf("expected first URL to be called once, got %d", calledFirstURL.Load())
	}
	if calledSecondURL.Load() != 0 {
		t.Errorf("expected second URL not to be called, got %d", calledSecondURL.Load())
	}
}

func TestUnexpose_FirstURLFails_SecondSucceeds(t *testing.T) {
	var calledFirstURL, calledSecondURL atomic.Int32

	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calledFirstURL.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv1.Close()

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledSecondURL.Add(1)
		assertHostPayload(t, r, "old.example.com")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv2.Close()

	overrideURLs(t, &removeHostURLs, []string{srv1.URL, srv2.URL})

	if err := unexpose("old.example.com"); err != nil {
		t.Fatalf("unexpose returned unexpected error: %v", err)
	}
	if calledFirstURL.Load() != 1 {
		t.Errorf("expected first URL to be called once, got %d", calledFirstURL.Load())
	}
	if calledSecondURL.Load() != 1 {
		t.Errorf("expected second URL to be called once, got %d", calledSecondURL.Load())
	}
}

func TestExpose_BothURLsFail(t *testing.T) {
	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv1.Close()

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv2.Close()

	overrideURLs(t, &addHostURLs, []string{srv1.URL, srv2.URL})

	if err := expose("fail.example.com"); err == nil {
		t.Fatal("expected error when both URLs fail, got nil")
	}
}

func TestUnexpose_BothURLsFail(t *testing.T) {
	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv1.Close()

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv2.Close()

	overrideURLs(t, &removeHostURLs, []string{srv1.URL, srv2.URL})

	if err := unexpose("fail.example.com"); err == nil {
		t.Fatal("expected error when both URLs fail, got nil")
	}
}

func overrideURLs(t *testing.T, target *[]string, urls []string) {
	t.Helper()
	orig := *target
	*target = urls
	t.Cleanup(func() { *target = orig })
}

func assertHostPayload(t *testing.T, r *http.Request, expectedHost string) {
	t.Helper()
	if r.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}
	var hosts []string
	if err := json.Unmarshal(body, &hosts); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}
	if len(hosts) != 1 || hosts[0] != expectedHost {
		t.Errorf("expected payload %q, got %v", []string{expectedHost}, hosts)
	}
}
