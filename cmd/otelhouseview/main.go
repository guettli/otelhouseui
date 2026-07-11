// Command otelhouseview runs the query workbench as a standalone binary.
//
// It is a thin wrapper over the explore package, mounted at "/". This is a
// convenience for local development, not the intended production shape: in
// production a host application (agentloop) imports explore and mounts it
// behind its own authentication. This binary authenticates nobody — do not put
// it on a public listener.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/guettli/otelhouseview/explore"
	"github.com/guettli/otelhouseview/internal/config"
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

	log.Printf("connecting to clickhouse, opening sqlite at %s", cfg.SQLitePath)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	svc, err := explore.New(ctx, explore.Config{
		DSN:       cfg.ClickHouseDSN,
		StorePath: cfg.SQLitePath,
		Prefix:    "/",
	})
	if err != nil {
		return err
	}
	defer func() { _ = svc.Close() }()

	addr := net.JoinHostPort("", strconv.Itoa(cfg.Port))
	srv := &http.Server{Addr: addr, Handler: svc.Handler(), ReadHeaderTimeout: 10 * time.Second}

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
