package static

import (
	"embed"
	"net/http"
)

// TODO: locally serve fonts, too?
//
//go:embed *.html *.css *.woff2 favicon.ico *.svg *.md
var FS embed.FS

// Server serves the html, css, woff2 (font)and ico embedded in FS.
var Server = http.FileServer(http.FS(FS))
