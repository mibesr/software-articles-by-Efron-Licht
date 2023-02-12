package main_test

import (
	"context"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	main "gitlab.com/efronlicht/blog/server"
	"gitlab.com/efronlicht/blog/server/static"
)

func TestMain(m *testing.M) {
	os.Setenv("PORT", "6483")
	go main.Run(context.Background())
	time.Sleep(50 * time.Millisecond)
	os.Exit(m.Run())
}

func TestUptime(t *testing.T) {
	if got := testGet(t, "debug/uptime"); !regexp.MustCompile(`\d+h \d+m \d+s`).MatchString(got) {
		t.Fatal(`expected \d+h \d+m \d+s: got `, got)
	}
}

func TestFiles(t *testing.T) {
	fs.WalkDir(static.FS, ".", func(path string, d fs.DirEntry, err error) error {
		log.Print(path)
		switch name := d.Name(); filepath.Ext(name) {
		case ".md", ".html", ".ico", ".woff2":
			testGet(t, name)
		}
		return nil
	})
}

func testGet(t *testing.T, p string) (body string) {
	t.Run(p, func(t *testing.T) {
		target := "http://localhost:6483/" + strings.TrimPrefix(p, "/")
		resp, err := http.Get(target)
		if err != nil {
			t.Fatalf("get %s: %v", target, err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("expected %d, got %d", 200, resp.StatusCode)
		}
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		body = string(b)
	})
	return body
}
