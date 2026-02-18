// Package embedded contiene los archivos estáticos del sitio web, embebidos en el binario de Go utilizando la directiva `embed`.
package embedded

import (
	"embed"
)

// WebFiles contiene el sitio web estático (HTML, CSS, JS)
//
//go:embed internal/assets/web
var WebFiles embed.FS
