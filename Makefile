# The Dagger pipeline in ci/ is the single source of truth for tests.
# `make ci` runs locally exactly what GitHub Actions runs, so a green local run
# implies a green CI run. Do not add a second test path (docker-compose etc).
#
# The pipeline needs a reachable Dagger engine. Point the Dagger SDK at a remote
# engine by exporting _EXPERIMENTAL_DAGGER_RUNNER_HOST before running; it is
# inherited by `go run` below. The cluster upload step is skipped unless
# OTELHOUSEUI_UPLOAD=1 (CI sets it only on pushes to main).

.PHONY: ci test ui-build ui-dev web web-dev app-test app-build
ci test:
	cd ci && go run .

# Build the static report locally against the committed sample data.
ui-build:
	cd ui && pnpm install --frozen-lockfile && pnpm run build

ui-dev:
	cd ui && pnpm install && pnpm run dev

# --- otelhouseview service (the vertical slice: Go binary + Svelte SPA) -------

# Build the SPA (Svelte + CodeMirror + ECharts) into web/build/ so `go build`
# can embed it. Run once before `app-build`; re-run after touching web/src.
web:
	cd web && pnpm install --frozen-lockfile && pnpm run build

# Dev server for the SPA. Assumes the Go service is running on :8080; Vite
# proxies /api and /healthz to it.
web-dev:
	cd web && pnpm install && pnpm run dev

# Go unit tests for the service module.
app-test:
	go test ./...

# Build the single self-contained binary. Requires `make web` first so the
# embedded SPA is up to date.
app-build:
	go build -o bin/otelhouseview ./cmd/otelhouseview
