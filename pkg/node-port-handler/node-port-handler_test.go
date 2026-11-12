package nodeporthandler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExpose_SendsTokenAndBody(t *testing.T) {
	t.Setenv("CRC_REST_API_TOKEN", "test-token")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		assertBody(t, r, map[string]string{"Local": "127.0.0.1:30080", "Remote": "192.168.127.2:30080"})
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	overrideURL(t, &exposeURL, srv.URL)

	if err := expose(30080); err != nil {
		t.Fatalf("expose returned unexpected error: %v", err)
	}
}

func TestUnexpose_SendsToken(t *testing.T) {
	t.Setenv("CRC_REST_API_TOKEN", "test-token")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "test-token")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	overrideURL(t, &unexposeURL, srv.URL)

	if err := unexpose(30080); err != nil {
		t.Fatalf("unexpose returned unexpected error: %v", err)
	}
}

func TestForwarderURLs(t *testing.T) {
	if exposeURL != "http://host/services/forwarder/expose" {
		t.Errorf("unexpected exposeURL %q", exposeURL)
	}
	if unexposeURL != "http://host/services/forwarder/unexpose" {
		t.Errorf("unexpected unexposeURL %q", unexposeURL)
	}
}

func overrideURL(t *testing.T, target *string, url string) {
	t.Helper()
	orig := *target
	*target = url
	t.Cleanup(func() { *target = orig })
}

func assertBearerToken(t *testing.T, r *http.Request, token string) {
	t.Helper()
	if auth := r.Header.Get("Authorization"); auth != "Bearer "+token {
		t.Errorf("expected Authorization %q, got %q", "Bearer "+token, auth)
	}
}

func assertBody(t *testing.T, r *http.Request, expected map[string]string) {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}
	var got map[string]string
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}
	if len(got) != len(expected) {
		t.Errorf("expected body %v, got %v", expected, got)
	}
	for k, v := range expected {
		if got[k] != v {
			t.Errorf("expected %s=%q, got %q", k, v, got[k])
		}
	}
}
