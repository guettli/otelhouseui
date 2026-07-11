package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/guettli/otelhouseview/internal/ch"
	"github.com/guettli/otelhouseview/internal/store"
)

type fakeCH struct {
	lastSQL    string
	lastParams map[string]any
	result     *ch.Result
	err        error
	pingErr    error
}

func (f *fakeCH) Query(_ context.Context, sql string, params map[string]any) (*ch.Result, error) {
	f.lastSQL = sql
	f.lastParams = params
	if f.err != nil {
		return nil, f.err
	}
	if f.result != nil {
		return f.result, nil
	}
	return &ch.Result{
		Columns: []ch.Column{{Name: "one", Type: "UInt8"}},
		Rows:    [][]any{{uint8(1)}},
	}, nil
}
func (f *fakeCH) Ping(context.Context) error { return f.pingErr }

func newTestServer(t *testing.T, exec QueryExecutor, web fs.FS) (*httptest.Server, *store.Store) {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if exec == nil {
		exec = &fakeCH{}
	}
	ts := httptest.NewServer(New(st, exec, web).Handler())
	t.Cleanup(ts.Close)
	return ts, st
}

func do(t *testing.T, method, url, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func decode(t *testing.T, res *http.Response, into any) {
	t.Helper()
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(into); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

func TestPostQuery_missingSQL(t *testing.T) {
	ts, _ := newTestServer(t, nil, nil)
	res := do(t, http.MethodPost, ts.URL+"/api/query", `{"sql":""}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}
	var body errBody
	decode(t, res, &body)
	if body.Error == "" {
		t.Error("expected error message")
	}
}

func TestPostQuery_success(t *testing.T) {
	fake := &fakeCH{}
	ts, _ := newTestServer(t, fake, nil)

	body := `{"sql":"SELECT 1","params":{"x":1}}`
	res := do(t, http.MethodPost, ts.URL+"/api/query", body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	var got ch.Result
	decode(t, res, &got)
	if len(got.Rows) != 1 {
		t.Errorf("rows = %d, want 1", len(got.Rows))
	}
	if fake.lastSQL != "SELECT 1" {
		t.Errorf("lastSQL = %q", fake.lastSQL)
	}
}

func TestSavedCRUD_and_UnknownType(t *testing.T) {
	ts, st := newTestServer(t, nil, nil)

	// Unknown type is rejected.
	bad := `{"name":"bad","sql_template":"SELECT 1","params":[{"name":"x","type":"NopeType"}]}`
	res := do(t, http.MethodPost, ts.URL+"/api/saved-queries", bad)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad type: status = %d, want 400", res.StatusCode)
	}

	// Valid create.
	body := `{"name":"q","sql_template":"SELECT {n:UInt32}","params":[{"name":"n","type":"UInt32"}]}`
	res = do(t, http.MethodPost, ts.URL+"/api/saved-queries", body)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create: status = %d, want 201", res.StatusCode)
	}
	var created store.SavedQuery
	decode(t, res, &created)

	// List.
	res = do(t, http.MethodGet, ts.URL+"/api/saved-queries", "")
	if res.StatusCode != 200 {
		t.Fatalf("list: status = %d", res.StatusCode)
	}
	var list []store.SavedQuery
	decode(t, res, &list)
	if len(list) != 1 {
		t.Errorf("list len = %d, want 1", len(list))
	}

	// Duplicate name → 409.
	res = do(t, http.MethodPost, ts.URL+"/api/saved-queries", body)
	if res.StatusCode != http.StatusConflict {
		t.Errorf("duplicate: status = %d, want 409", res.StatusCode)
	}

	// Delete.
	url := ts.URL + "/api/saved-queries/" + itoa(created.ID)
	res = do(t, http.MethodDelete, url, "")
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: status = %d, want 204", res.StatusCode)
	}
	if _, err := st.Get(context.Background(), created.ID); err == nil {
		t.Error("expected row to be gone")
	}
}

func TestRunSaved_bindsTypedParams(t *testing.T) {
	fake := &fakeCH{}
	ts, st := newTestServer(t, fake, nil)

	q := &store.SavedQuery{
		Name:        "run-me",
		SQLTemplate: "SELECT {n:UInt32}, {s:String}",
		Params: []store.Param{
			{Name: "n", Type: "UInt32"},
			{Name: "s", Type: "String"},
		},
	}
	if err := st.Insert(context.Background(), q); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	body := `{"params":{"n":42,"s":"hello"}}`
	res := do(t, http.MethodPost, ts.URL+"/api/saved-queries/"+itoa(q.ID)+"/run", body)
	if res.StatusCode != 200 {
		t.Fatalf("status = %d, body=%s", res.StatusCode, readBody(res))
	}
	if got, want := fake.lastParams["n"], uint32(42); got != want {
		t.Errorf("n = %v (%T), want %v", got, got, want)
	}
	if got := fake.lastParams["s"]; got != "hello" {
		t.Errorf("s = %v", got)
	}
}

func TestHealthz_okAndFailure(t *testing.T) {
	fake := &fakeCH{}
	ts, _ := newTestServer(t, fake, nil)

	res := do(t, http.MethodGet, ts.URL+"/healthz", "")
	if res.StatusCode != 200 {
		t.Errorf("healthy: status = %d", res.StatusCode)
	}
	fake.pingErr = errString("boom")
	res = do(t, http.MethodGet, ts.URL+"/healthz", "")
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("unhealthy: status = %d, want 503", res.StatusCode)
	}
}

func TestSPAFallback(t *testing.T) {
	web := fstest.MapFS{
		"index.html":    {Data: []byte("<html>root</html>")},
		"assets/app.js": {Data: []byte("console.log(1)")},
	}
	ts, _ := newTestServer(t, nil, web)

	// Existing asset served as-is.
	res := do(t, http.MethodGet, ts.URL+"/assets/app.js", "")
	if got := readBody(res); got != "console.log(1)" {
		t.Errorf("asset body = %q", got)
	}
	// Client-side route falls back to index.html.
	res = do(t, http.MethodGet, ts.URL+"/saved/42", "")
	if got := readBody(res); !strings.Contains(got, "root") {
		t.Errorf("fallback body = %q, want index.html", got)
	}
	// Unknown /api route stays a JSON 404.
	res = do(t, http.MethodGet, ts.URL+"/api/nope", "")
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("unknown api: status = %d, want 404", res.StatusCode)
	}
}

func itoa(i int64) string { return strconv.FormatInt(i, 10) }

func readBody(res *http.Response) string {
	defer res.Body.Close()
	buf := &bytes.Buffer{}
	_, _ = buf.ReadFrom(res.Body)
	return buf.String()
}

type errString string

func (e errString) Error() string { return string(e) }
