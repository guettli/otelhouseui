# The Dagger pipeline in ci/ is the single source of truth for tests.
# `make ci` runs locally exactly what GitHub Actions runs, so a green local run
# implies a green CI run. Do not add a second test path (docker-compose etc).
#
# The pipeline needs a reachable Dagger engine. Point the Dagger SDK at a remote
# engine by exporting _EXPERIMENTAL_DAGGER_RUNNER_HOST before running; it is
# inherited by `go run` below. The cluster upload step is skipped unless
# OTELHOUSEUI_UPLOAD=1 (CI sets it only on pushes to main).

.PHONY: ci test ui-build ui-dev
ci test:
	cd ci && go run .

# Build the static report locally against the committed sample data.
ui-build:
	cd ui && pnpm install --frozen-lockfile && pnpm run build

ui-dev:
	cd ui && pnpm install && pnpm run dev
