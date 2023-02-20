// rendermarkdown searches a directory for markdown files and renders them as HTML to the output directory.
// // USAGE:
// // rendermarkdown SRC DST
package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/PuerkitoBio/goquery"
	"github.com/sourcegraph/syntaxhighlight"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
)

func must[T any](t T, err error) T {
	if err != nil {
		_, f, line, _ := runtime.Caller(1)
		fmt.Fprintf(os.Stderr, "%s %d: fatal err: %v\n", f, line, err)
		os.Exit(1)
	}
	return t
}

func main() {
	log.SetPrefix("rendermd\t")

	if len(os.Args) != 3 {
		log.Print("expected two command-line arguments")
		log.Fatal("USAGE: rendermd srcdir dstdir")
	}
	srcDir, dstDir := must(filepath.Abs(os.Args[1])), must(filepath.Abs(os.Args[2]))
	must(0, os.MkdirAll(dstDir, 0o777))
	log.Println("srcDir: ", srcDir)
	log.Println("dstDir: ", dstDir)
	const format = "rendermd\t%s\t->\t%s\n"
	log.Println("scanning...")
	var wg sync.WaitGroup
	type res struct {
		md, html string
	}
	ch := make(chan res, 24)
	walkFunc := func(srcPath string, d fs.DirEntry, err error) error {
		tw := tabwriter.NewWriter(os.Stderr, 2, 2, 2, ' ', 0)

		defer tw.Flush()
		if strings.Contains(srcPath, "vendor") || !strings.Contains(srcPath, "efronlicht") || strings.Contains(srcPath, dstDir) {
			return fs.SkipDir
		}
		_ = must(0, err)
		if filepath.Ext(srcPath) != ".md" {
			return nil
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			dstPath := strings.ReplaceAll(filepath.Join(dstDir, filepath.Base(srcPath)), ".md", ".html")

			fmt.Fprintf(tw, format, srcPath, dstPath)
			must(0, os.WriteFile(dstPath, renderMarkdown(srcPath), 0o777))
			ch <- res{md: srcPath, html: dstPath}
		}()
		return nil
	}

	err := filepath.WalkDir(srcDir, walkFunc)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()
	tw := tabwriter.NewWriter(os.Stderr, 2, 2, 2, ' ', 0)
	fmt.Fprintf(tw, format, "src", "dst")
	fmt.Fprintf(tw, format, strings.Repeat("-", 20), strings.Repeat("-", 20))
	defer tw.Flush()
	for r := range ch {
		fmt.Fprintf(tw, format, r.md, r.html)
	}
}

func renderMarkdown(path string) []byte {
	renderer := html.NewRenderer(html.RendererOptions{
		Icon:           "/favicon.ico",
		AbsolutePrefix: "",
		CSS:            "/s.css",
		Flags:          html.CommonFlags | html.CompletePage,
		Title:          strings.TrimSuffix(filepath.Base(path), ".md"),
	})
	html := markdown.ToHTML(markdown.NormalizeNewlines(must(os.ReadFile(path))), nil, renderer)
	doc := must(goquery.NewDocumentFromReader(bytes.NewReader(html)))
	// find code-parts via css selector and replace them with highlighted versions
	doc.Find("code[class*=\"language-\"]").Each(func(i int, s *goquery.Selection) {
		oldCode := s.Text()
		s.SetHtml(string(must(syntaxhighlight.AsHTML([]byte(oldCode)))))
	})
	html = []byte((must(doc.Html())))
	html = bytes.ReplaceAll(html, []byte("<html><head></head><body>"), nil)
	html = bytes.ReplaceAll(html, []byte("</body></html>"), nil)
	return html
}
