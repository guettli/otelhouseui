// Command otelhouseview runs the query + visualise UI as a single binary.
//
// The binary bundles the SPA (Svelte + CodeMirror + ECharts), talks to a
// read-only ClickHouse user for all queries, and persists saved query
// templates in a local SQLite file.
package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	otelhouseview "github.com/guettli/otelhouseview"
	"github.com/guettli/otelhouseview/internal/ch"
	"github.com/guettli/otelhouseview/internal/config"
	"github.com/guettli/otelhouseview/internal/httpapi"
	"github.com/guettli/otelhouseview/internal/starters"
	"github.com/guettli/otelhouseview/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log.Printf("connecting to clickhouse")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	chc, err := ch.Open(ctx, cfg.ClickHouseDSN)
	if err != nil {
		return fmt.Errorf("clickhouse: %w", err)
	}
	defer func() { _ = chc.Close() }()

	log.Printf("opening sqlite at %s", cfg.SQLitePath)
	st, err := store.Open(cfg.SQLitePath)
	if err != nil {
		return fmt.Errorf("sqlite: %w", err)
	}
	defer func() { _ = st.Close() }()

	if err := starters.Seed(context.Background(), st); err != nil {
		return fmt.Errorf("seed starters: %w", err)
	}

	web, err := fs.Sub(otelhouseview.WebFS(), "web/build")
	if err != nil {
		return fmt.Errorf("embed sub: %w", err)
	}

	handler := httpapi.New(st, chc, web).Handler()
	addr := net.JoinHostPort("", strconv.Itoa(cfg.Port))
	srv := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second}

	stopped := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", addr)
		stopped <- srv.ListenAndServe()
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case s := <-sig:
		log.Printf("received %s, shutting down", s)
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		return srv.Shutdown(shutCtx)
	case err := <-stopped:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
