// prezip walks a directory recursively, combining non-zipped files into an archive and omitting them to stdout.
// it uses DEFLATE on most file types, but
// just stores files that are already compressed (.png, .woff2 .jpg; perhaps more later )
//
//	usage:
//	   prezip DIR
package main

import (
	"archive/zip"
	"io"
	"io/fs"
	"log"

	"os"
	"path/filepath"
	"strings"
)

func main() {
	dir := must(filepath.Abs(os.Args[1]))
	log.SetPrefix("prezip")

	f := must(os.Create(filepath.Join(dir, "assets.zip")))
	zw := zip.NewWriter(os.Stdout)
	var files, bytes int64
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if strings.Contains(d.Name(), ".zip") || strings.Contains(d.Name(), ".gz") || d.IsDir() {
			return nil
		}

		src := must(os.Open(filepath.Join(dir, d.Name())))
		var dst io.Writer
		switch filepath.Ext(src.Name()) {
		case ".woff2", ".png", ".jpg": // already compressed; a layer of deflate won't help.
			header := must(zip.FileInfoHeader(must(d.Info())))
			dst = must(zw.CreateRaw(header))
		default:
			dst = must(zw.Create(d.Name()))
		}
		bytes += must(io.Copy(dst, src))
		files++
		src.Close()
		return nil
	})
	if err != nil {
		panic(err)
	}
	zw.Close()
	f.Close()

	log.Printf("combined %d files (%04d KiB)", files, bytes)
}
func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}
