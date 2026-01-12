// Package assets contains embedded web files for the test client.
package assets

import "embed"

//go:embed web/*
var WebFiles embed.FS
