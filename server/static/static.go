package static

import (
	"embed"
	"net/http"
)

// FS embeds the html, css, wof2, svg, ico, and md files in this directory.
//
//go:embed *.html *.css *.woff2 favicon.ico *.svg *.md
var FS embed.FS

// Server serves the items embedded in FS.
var Server = http.FileServer(http.FS(FS))
