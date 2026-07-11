package explore

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/guettli/otelhouseview/explore/internal/ch"
	"github.com/guettli/otelhouseview/explore/internal/httpapi"
	"github.com/guettli/otelhouseview/explore/internal/store"
)

// fakeCH stands in for ClickHouse: New() dials for real, so the mount tests
// build the Service's parts directly instead.
type fakeCH struct{}

func (fakeCH) Query(context.Context, string, map[string]any) (*ch.Result, error) {
	return &ch.Result{Columns: []ch.Column{{Name: "one", Type: "UInt8"}}, Rows: [][]any{{uint8(1)}}}, nil
}
func (fakeCH) Ping(context.Context) error { return nil }

func newTestService(t *testing.T, prefix string) *Service {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	web, err := webBuild()
	if err != nil {
		t.Fatalf("webBuild: %v", err)
	}
	base := normalizeBase(prefix)
	return &Service{
		store:   st,
		handler: mount(base, httpapi.New(st, fakeCH{}, web, base).Handler()),
	}
}

func get(t *testing.T, ts *httptest.Server, path string) (int, string) {
	t.Helper()
	res, err := ts.Client().Get(ts.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return res.StatusCode, string(body)
}

// TestMountedUnderPrefix is the contract with the host app (agentloop mounts
// explore at /explore behind its own auth): with the host stripping the prefix,
// the SPA, its assets and the JSON API must all still resolve, and the served
// HTML must carry the mount base so the SPA knows where to send its fetches.
func TestMountedUnderPrefix(t *testing.T) {
	svc := newTestService(t, "/explore")

	mux := http.NewServeMux()
	mux.Handle("/explore/", http.StripPrefix("/explore", svc.Handler()))
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "host app", http.StatusTeapot) // proves we never leak out of the mount
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	// (a) the SPA index is served at the mount point.
	code, body := get(t, ts, "/explore/")
	if code != http.StatusOK {
		t.Fatalf("GET /explore/ = %d, want 200", code)
	}
	if !strings.Contains(body, `<div id="app">`) {
		t.Fatalf("GET /explore/ did not serve the SPA index:\n%s", body)
	}

	// (b) an API route responds under the prefix.
	code, body = get(t, ts, "/explore/api/saved-queries")
	if code != http.StatusOK {
		t.Fatalf("GET /explore/api/saved-queries = %d (%s), want 200", code, body)
	}
	if !strings.HasPrefix(strings.TrimSpace(body), "[") {
		t.Fatalf("saved-queries body is not a JSON array: %s", body)
	}

	// (c) the injected base appears in the HTML, and no unsubstituted
	//     placeholder or root-absolute asset URL survives — either would make
	//     the SPA fetch against the host app instead of against us.
	_, index := get(t, ts, "/explore/")
	if !strings.Contains(index, `window.__EXPLORE_BASE__ = '/explore/'`) {
		t.Errorf("index.html does not carry the mount base:\n%s", index)
	}
	if strings.Contains(index, httpapi.BasePlaceholder) {
		t.Errorf("index.html still contains the unsubstituted placeholder %q:\n%s", httpapi.BasePlaceholder, index)
	}
	if strings.Contains(index, `src="/assets/`) || strings.Contains(index, `href="/assets/`) {
		t.Errorf("index.html references root-absolute assets, which 404 under a mount:\n%s", index)
	}
}

// TestMountedWithoutStripPrefix covers the other wiring a host might pick: the
// handler tolerates requests that still carry the prefix.
func TestMountedWithoutStripPrefix(t *testing.T) {
	svc := newTestService(t, "/explore")
	ts := httptest.NewServer(svc.Handler())
	t.Cleanup(ts.Close)

	for _, path := range []string{"/explore", "/explore/", "/explore/#/saved"} {
		if code, _ := get(t, ts, path); code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, code)
		}
	}
	if code, body := get(t, ts, "/explore/api/saved-queries"); code != http.StatusOK {
		t.Errorf("GET /explore/api/saved-queries = %d (%s), want 200", code, body)
	}
}

// TestRootMount is the standalone binary's shape.
func TestRootMount(t *testing.T) {
	svc := newTestService(t, "/")
	ts := httptest.NewServer(svc.Handler())
	t.Cleanup(ts.Close)

	code, body := get(t, ts, "/")
	if code != http.StatusOK || !strings.Contains(body, `<div id="app">`) {
		t.Fatalf("GET / = %d, body:\n%s", code, body)
	}
	if !strings.Contains(body, `window.__EXPLORE_BASE__ = '/'`) {
		t.Errorf("index.html does not carry the root base:\n%s", body)
	}
	if code, body := get(t, ts, "/healthz"); code != http.StatusOK {
		t.Errorf("GET /healthz = %d (%s), want 200", code, body)
	}
}

func TestNormalizeBase(t *testing.T) {
	for in, want := range map[string]string{
		"":          "/",
		"/":         "/",
		"explore":   "/explore/",
		"/explore":  "/explore/",
		"/explore/": "/explore/",
		"/a/b":      "/a/b/",
	} {
		if got := normalizeBase(in); got != want {
			t.Errorf("normalizeBase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNewRequiresConfig(t *testing.T) {
	if _, err := New(context.Background(), Config{StorePath: ":memory:"}); err == nil {
		t.Error("New without DSN: want error")
	}
	if _, err := New(context.Background(), Config{DSN: "clickhouse://x"}); err == nil {
		t.Error("New without StorePath: want error")
	}
}
