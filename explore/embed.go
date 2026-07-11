package explore

import "embed"

// The built SPA is committed to the repository, not built on the fly: this
// package is imported as a library, and `go get` ships exactly what go:embed
// can see in the module. Rebuild with `make web` (or `cd explore/web && pnpm
// run build`) and commit explore/web/build/ whenever explore/web/src changes.
//
//go:embed all:web/build
var webFS embed.FS
