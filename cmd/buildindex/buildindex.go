package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	log.SetPrefix("buildindex\t")
	if len(os.Args) != 2 {
		log.Fatal("expected exactly one command-line argument\nusage:\tbuildindex DIR")
	}
	html := []byte(`<!DOCTYPE html><html><head>
	<title>index.html</title>
	<meta charset="utf-8"/>
	<link rel="stylesheet" type="text/css" href="/s.css"/>
	</head>
	<body>
	<h1> articles </h1>
`)

	dir := must(filepath.Abs(os.Args[1]))
	for _, e := range must(must(os.Open(dir)).ReadDir(-1)) {
		if n := e.Name(); strings.Contains(filepath.Ext(n), "html") {
			html = fmt.Appendf(html, `<h4><a href="/%s">%s</a>`+"\n</h4>", n, n)
		}
	}
	html = append(html, "</body>"...)
	dst := filepath.Join(dir, "index.html")
	os.WriteFile(dst, html, 0o777)
	log.Printf("wrote %s", dst)

}

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}
