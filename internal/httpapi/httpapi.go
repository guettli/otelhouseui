// Package httpapi wires HTTP routes for the otelhouseview service.
//
// The API is deliberately signal-agnostic: one query executor + one saved-
// query CRUD is enough to serve metrics charts, log-volume-over-time and span
// rate/latency without per-signal endpoints. All ClickHouse traffic goes
// through a QueryExecutor interface so tests can substitute a fake.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"github.com/guettli/otelhouseview/internal/ch"
	"github.com/guettli/otelhouseview/internal/store"
)

// QueryExecutor is the small subset of the ClickHouse client this package
// depends on. Tests provide a fake implementation.
type QueryExecutor interface {
	Query(ctx context.Context, sql string, params map[string]any) (*ch.Result, error)
	Ping(ctx context.Context) error
}

// Server is the concrete HTTP handler.
type Server struct {
	store *store.Store
	ch    QueryExecutor
	web   fs.FS
}

// New builds a Server. `web` should be the SPA build tree (rooted so that
// "index.html" is at the top level). Pass nil to disable the SPA fallback,
// which is convenient in tests.
func New(s *store.Store, exec QueryExecutor, web fs.FS) *Server {
	return &Server{store: s, ch: exec, web: web}
}

// Handler returns the wired HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("POST /api/query", s.postQuery)

	mux.HandleFunc("GET /api/saved-queries", s.listSaved)
	mux.HandleFunc("POST /api/saved-queries", s.createSaved)
	mux.HandleFunc("GET /api/saved-queries/{id}", s.getSaved)
	mux.HandleFunc("PUT /api/saved-queries/{id}", s.updateSaved)
	mux.HandleFunc("DELETE /api/saved-queries/{id}", s.deleteSaved)
	mux.HandleFunc("POST /api/saved-queries/{id}/run", s.runSaved)

	mux.HandleFunc("/", s.spa)

	return mux
}

// --- JSON helpers -----------------------------------------------------------

type errBody struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, errBody{Error: err.Error()})
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

// --- /healthz ---------------------------------------------------------------

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		writeErr(w, http.StatusServiceUnavailable, fmt.Errorf("sqlite: %w", err))
		return
	}
	if err := s.ch.Ping(r.Context()); err != nil {
		writeErr(w, http.StatusServiceUnavailable, fmt.Errorf("clickhouse: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- /api/query -------------------------------------------------------------

type queryReq struct {
	SQL    string         `json:"sql"`
	Params map[string]any `json:"params,omitempty"`
}

func (s *Server) postQuery(w http.ResponseWriter, r *http.Request) {
	var req queryReq
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.SQL) == "" {
		writeErr(w, http.StatusBadRequest, errors.New("sql is required"))
		return
	}
	res, err := s.ch.Query(r.Context(), req.SQL, req.Params)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// --- /api/saved-queries CRUD -----------------------------------------------

type savedReq struct {
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	SQLTemplate string        `json:"sql_template"`
	Params      []store.Param `json:"params,omitempty"`
	DefaultViz  string        `json:"default_viz,omitempty"`
}

func (s *Server) listSaved(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.List(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if list == nil {
		list = []store.SavedQuery{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) getSaved(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	q, err := s.store.Get(r.Context(), id)
	if err != nil {
		s.writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, q)
}

func (s *Server) createSaved(w http.ResponseWriter, r *http.Request) {
	var req savedReq
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err := validateSaved(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	q := &store.SavedQuery{
		Name:        req.Name,
		Description: req.Description,
		SQLTemplate: req.SQLTemplate,
		Params:      req.Params,
		DefaultViz:  req.DefaultViz,
	}
	if err := s.store.Insert(r.Context(), q); err != nil {
		s.writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, q)
}

func (s *Server) updateSaved(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	var req savedReq
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err := validateSaved(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	q := &store.SavedQuery{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		SQLTemplate: req.SQLTemplate,
		Params:      req.Params,
		DefaultViz:  req.DefaultViz,
	}
	if err := s.store.Update(r.Context(), q); err != nil {
		s.writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, q)
}

func (s *Server) deleteSaved(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err := s.store.Delete(r.Context(), id); err != nil {
		s.writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- /api/saved-queries/{id}/run -------------------------------------------

type runReq struct {
	Params map[string]any `json:"params,omitempty"`
}

func (s *Server) runSaved(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	q, err := s.store.Get(r.Context(), id)
	if err != nil {
		s.writeStoreErr(w, err)
		return
	}

	var req runReq
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
	}
	bound, err := bindParams(q.Params, req.Params)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	res, err := s.ch.Query(r.Context(), q.SQLTemplate, bound)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// --- SPA fallback ----------------------------------------------------------

func (s *Server) spa(w http.ResponseWriter, r *http.Request) {
	// Anything under /api/ that reaches "/" is a routing mistake, not a page.
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeErr(w, http.StatusNotFound, errors.New("no such route"))
		return
	}
	if s.web == nil {
		http.NotFound(w, r)
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" {
		name = "index.html"
	}
	// Serve the requested asset if it exists, otherwise fall back to index.html
	// so client-side routing (/saved, /saved/{id}) works from a fresh page load.
	f, err := s.web.Open(name)
	if err != nil {
		f, err = s.web.Open("index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		name = "index.html"
	}
	_ = f.Close()

	http.ServeFileFS(w, r, s.web, name)
}

// --- helpers ---------------------------------------------------------------

func pathID(r *http.Request) (int64, error) {
	raw := r.PathValue("id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid id %q", raw)
	}
	return id, nil
}

func validateSaved(req *savedReq) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(req.SQLTemplate) == "" {
		return errors.New("sql_template is required")
	}
	for i, p := range req.Params {
		if p.Name == "" {
			return fmt.Errorf("params[%d].name is required", i)
		}
		if !SupportedParamType(p.Type) {
			return fmt.Errorf("params[%d].type %q is not supported", i, p.Type)
		}
	}
	return nil
}

func (s *Server) writeStoreErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, http.StatusNotFound, err)
	case errors.Is(err, store.ErrDuplicateName):
		writeErr(w, http.StatusConflict, err)
	default:
		writeErr(w, http.StatusInternalServerError, err)
	}
}
