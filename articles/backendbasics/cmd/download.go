// download is a command-line tool to download a file from a URL.
// usage: download [-timeout duration] url filename
package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	dir := flag.String("dir", ".", "directory to save file")
	timeout := flag.Duration("timeout", 30*time.Second, "timeout for download")
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		log.Fatal("usage: download [-timeout duration] url filename")
	}
	url, filename := args[0], args[1]
	// always set a timeout when you make an HTTP request.
	c := http.Client{Timeout: *timeout}

	// always use context when you make an HTTP request; if you don't know which to use, use context.TODO().
	// we'll talk about contexts later in this article.
	req, err := http.NewRequestWithContext(context.TODO(), "GET", url, nil)
	if err != nil {
		log.Fatalf("creating request: GET %q: %v", url, err)
	}
	resp, err := c.Do(req)
	if err != nil {
		log.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("response status: %s", resp.Status)
	}
	dst := filepath.Join(*dir, filename)
	f, err := os.Create(dst)
	if err != nil {
		log.Fatalf("creating file: %v", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		log.Fatalf("copying response to file: %v", err)
	}

}
