// Package explore is the otelhouseview SQL workbench packaged as an embeddable
// sub-application: one http.Handler that serves both its JSON API and its own
// embedded Svelte SPA, mountable under any prefix inside a host Go service.
//
// # It does no authentication, on purpose
//
// The handler returned by [Service.Handler] executes ARBITRARY SQL sent by
// ARBITRARY callers. It has no login, no session, no CSRF check, no rate limit
// and it will never grow one. Mounting it on a public listener without wrapping
// it hands the world a SQL console.
//
//	http.Handle("/explore/", host.RequireLogin(http.StripPrefix("/explore", svc.Handler())))
//
// The host owns authentication. That is not laziness, it is the only shape that
// composes: the host (today: agentloop) already has passkey/WebAuthn sessions,
// an ingress and a TLS cert; a second auth stack per tenant would be a second
// thing to get wrong.
//
// # The security boundary is the ClickHouse identity in the DSN
//
// [Config.DSN] must already be scoped to exactly the data its users may see. In
// the otelhouse deployment that means a per-tenant `<ns>_ro` ClickHouse user
// with row policies pinned to `ResourceAttributes['tenant'] = '<ns>'`, plus a
// server-side settings profile capping query cost (max_execution_time,
// max_rows_to_read, max_bytes_to_read, max_result_rows, max_memory_usage).
//
// Given that, handing a tenant a raw SQL prompt is safe by construction: the
// worst SQL they can write still cannot read another tenant's rows and still
// cannot outrun the profile's limits. This package therefore adds NO tenancy
// predicate and NO Go-side limits of its own — either would look like a
// security boundary without being one, and would drift from the real one.
//
// # Mounting
//
// The host is expected to strip the prefix ([http.StripPrefix]); the handler
// routes on paths relative to its mount point ("/", "/healthz", "/api/...").
// [Config.Prefix] is only used to generate correct absolute URLs (assets, API
// base) in the served index.html. As a convenience the handler also tolerates
// the un-stripped case: if a request path still starts with Config.Prefix, it
// strips it itself. Getting Config.Prefix wrong is what breaks — not the choice
// of who strips.
package explore

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/guettli/otelhouseview/explore/internal/ch"
	"github.com/guettli/otelhouseview/explore/internal/httpapi"
	"github.com/guettli/otelhouseview/explore/internal/starters"
	"github.com/guettli/otelhouseview/explore/internal/store"
)

// Config configures a workbench instance.
type Config struct {
	// DSN is a clickhouse-go v2 DSN, e.g. "clickhouse://acme_ro:***@ch:9000/otel".
	// Its ClickHouse identity IS the security boundary: every query a user types
	// runs as this user, under its row policies and its settings profile.
	// Required.
	DSN string

	// StorePath is the SQLite file holding saved queries. The caller owns the
	// path (and its lifetime, backups, and per-host uniqueness). ":memory:"
	// gives an ephemeral store. Required.
	StorePath string

	// Prefix is the path the host mounts this handler under, e.g. "/explore".
	// "" or "/" means the handler owns the root. It is used ONLY to build
	// absolute URLs in the served HTML — see the package doc on mounting.
	Prefix string
}

// Service owns the ClickHouse connection, the SQLite store and the HTTP
// handler. Create with [New]; release with [Close].
type Service struct {
	ch      *ch.Client
	store   *store.Store
	handler http.Handler
}

// New dials ClickHouse, opens (and migrates) the SQLite store, seeds the
// starter queries and wires the handler. ctx bounds the setup only, not the
// lifetime of the service.
func New(ctx context.Context, cfg Config) (*Service, error) {
	if cfg.DSN == "" {
		return nil, errors.New("explore: Config.DSN is required")
	}
	if cfg.StorePath == "" {
		return nil, errors.New("explore: Config.StorePath is required")
	}

	chc, err := ch.Open(ctx, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("explore: clickhouse: %w", err)
	}
	st, err := store.Open(cfg.StorePath)
	if err != nil {
		_ = chc.Close()
		return nil, fmt.Errorf("explore: sqlite: %w", err)
	}
	if err := starters.Seed(ctx, st); err != nil {
		_ = st.Close()
		_ = chc.Close()
		return nil, fmt.Errorf("explore: seed starters: %w", err)
	}

	web, err := webBuild()
	if err != nil {
		_ = st.Close()
		_ = chc.Close()
		return nil, err
	}

	base := normalizeBase(cfg.Prefix)
	return &Service{
		ch:      chc,
		store:   st,
		handler: mount(base, httpapi.New(st, chc, web, base).Handler()),
	}, nil
}

// Handler returns the workbench handler: the JSON API under /api/, a /healthz
// probe, and the SPA (with a fallback to index.html) on everything else.
//
// It is UNAUTHENTICATED. Wrap it. See the package doc.
func (s *Service) Handler() http.Handler { return s.handler }

// Close releases the SQLite and ClickHouse handles.
func (s *Service) Close() error {
	return errors.Join(s.store.Close(), s.ch.Close())
}

// webBuild returns the embedded SPA build tree rooted at index.html.
func webBuild() (fs.FS, error) {
	sub, err := fs.Sub(webFS, "web/build")
	if err != nil {
		return nil, fmt.Errorf("explore: embedded SPA: %w", err)
	}
	return sub, nil
}

// normalizeBase turns a mount prefix into a URL base: always one leading and
// one trailing slash ("/" for the root mount, "/explore/" for "/explore").
func normalizeBase(prefix string) string {
	p := strings.TrimSpace(prefix)
	p = strings.Trim(p, "/")
	if p == "" {
		return "/"
	}
	return "/" + p + "/"
}

// mount adapts the inner handler to hosts that do NOT strip the mount prefix.
// When the host does strip it (the documented shape), request paths never carry
// the prefix and this is a pass-through — except for the empty path that
// http.StripPrefix produces for a request to the bare mount point, which is
// normalized to "/".
func mount(base string, next http.Handler) http.Handler {
	bare := strings.TrimSuffix(base, "/") // "" for the root mount
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case bare != "" && (p == bare || strings.HasPrefix(p, base)):
			next.ServeHTTP(w, rewritePath(r, "/"+strings.TrimPrefix(strings.TrimPrefix(p, bare), "/")))
		case p == "":
			next.ServeHTTP(w, rewritePath(r, "/"))
		default:
			next.ServeHTTP(w, r)
		}
	})
}

// rewritePath clones r with a new URL path. RawPath is dropped so that
// EscapedPath re-derives from Path.
func rewritePath(r *http.Request, path string) *http.Request {
	r2 := r.Clone(r.Context())
	r2.URL.Path = path
	r2.URL.RawPath = ""
	return r2
}
