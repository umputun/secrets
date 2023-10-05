// Package ui provides embedded static files
package ui

import "embed"

// Files is embedded static files
//
//go:embed "html" "static"
var Files embed.FS
