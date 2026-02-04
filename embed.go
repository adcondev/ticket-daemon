package embedded

import (
	"embed"
	_ "embed"
)

// WebFiles contiene el sitio web estático (HTML, CSS, JS)
//
//go:embed internal/assets/web
var WebFiles embed.FS
