// Package assets provides embedded static files and templates
package assets

import "embed"

// Files is embedded static files
//
//go:embed "html" "static"
var Files embed.FS
