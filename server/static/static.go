package static

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

//go:embed assets.zip
var zipped []byte

var (
	FS    *zip.Reader
	files map[string]*zip.File
)

func init() {
	var err error
	FS, err = zip.NewReader(bytes.NewReader(zipped), int64(len(zipped)))
	if err != nil {
		panic("failed to read zipped file: " + err.Error())
	}
	files = make(map[string]*zip.File, len(FS.File))
	for _, f := range FS.File {
		files[f.Name] = f
	}
}

func ServeFile(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	if _, ok := files[path+".html"]; ok { // they forgot to add .html: show them where to find it.
		http.Redirect(w, r, "./"+path+".html", http.StatusPermanentRedirect)
		return
	}
	f, ok := files[path]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	// best-case scenario: just forward them the compressed file.
	if strings.Contains(r.Header.Get("Accept-Encoding"), "deflate") && f.Method == zip.Deflate {

		w.Header().Set("Content-Encoding", "deflate")
		if _, err := (io.Copy(w, must(f.OpenRaw()))); err != nil {
			zap.L().Error("failed to copy file", zap.Error(err), zap.String("file", f.Name))
		}
		return
	}
	if _, err := io.Copy(w, must(f.Open())); err != nil {
		zap.L().Error("failed to copy file", zap.Error(err), zap.String("file", f.Name))
	}
}

func must[T any](t T, err error) T {
	if err != nil {
		zap.L().Panic("fatal error", zap.Error(err))
	}
	return t
}
