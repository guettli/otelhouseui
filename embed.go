// Package otelhouseui exposes the embedded SPA build to the binary at
// cmd/otelhouseui. The embed lives here because go:embed can only reach files
// at or below the source file's directory, and the SPA sources live under
// web/, which is a sibling of internal/httpapi.
package otelhouseui

import "embed"

//go:embed all:web/build
var webFS embed.FS

// WebFS returns the embedded SPA build tree rooted at "web/build".
func WebFS() embed.FS { return webFS }
