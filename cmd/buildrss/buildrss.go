package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/html"
)

var srcDir, cacheDir string

func main() {
	if len(os.Args) < 3 {
		log.Fatal("expected two arguments (src cache)")
	}
	srcDir, cacheDir := must(filepath.Abs(os.Args[1])), must(filepath.Abs(os.Args[2]))

	must(0, os.MkdirAll(cacheDir, 0o777))
	log.SetPrefix("buildrss\t")
	log.Println("\tsrcDir\t", srcDir)
	log.Println("\tdstDir\t", cacheDir)
	today := time.Now()
	items := fromFile[map[string]Item](filepath.Join(cacheDir, "items.json"))
	if items == nil {
		items = make(map[string]Item)
	}
	checksums := fromFile[map[string][16]byte](filepath.Join(cacheDir, "checksums.json"))
	if checksums == nil {
		checksums = make(map[string][16]byte)
	}
	var changed int
	walkFunc := func(srcPath string, d fs.DirEntry, err error) error {
		if filepath.Ext(srcPath) != ".html" {
			return nil
		}

		log.SetPrefix("buildrss\t" + d.Name() + "\t")
		b := must(os.ReadFile(srcPath))
		wantSum := md5.Sum(b)
		log.Printf("calculated md5 of %s: %x", d.Name(), wantSum)
		if gotSum, ok := checksums[d.Name()]; ok && gotSum == wantSum {
			log.Println("checksum match; skipping")
			return nil // no need to update.
		}
		log.Println("checksum mismatch: updating")
		title, ok := findTitle(must(html.Parse(bytes.NewReader(b))))
		if !ok {
			panic(fmt.Errorf("no title for document %s", d.Name()))
		}
		items[d.Name()] = Item{
			Title:   title,
			GUID:    uuid.New(),
			Link:    fmt.Sprintf("https://eblog.fly.dev/%s", d.Name()),
			PubDate: today,
		}
		checksums[d.Name()] = wantSum
		changed++
		return nil
	}
	if err := filepath.WalkDir(srcDir, walkFunc); err != nil {
		panic(err)
	}
	if changed == 0 {
		os.Exit(0)
	}
	toFile("items.json", items)
	toFile("checksums.json", checksums)
}

func findTitle(n *html.Node) (string, bool) {
	if n.Type == html.ElementNode && n.Data == "title" {
		return n.FirstChild.Data, true
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if title, ok := findTitle(c); ok {
			return title, true
		}
	}
	return "", false
}

func toFile[T any](path string, t T) {
	b := must(json.MarshalIndent(t, "", "\t"))
	if err := os.WriteFile(path, b, 0o755); err != nil {
		panic(fmt.Errorf("writing %s to %v", b, path))
	}
}

func fromFile[T any](path string) T {
	var t T
	b, err := (os.ReadFile(path))
	if errors.Is(err, os.ErrNotExist) {
		log.Printf("file %s not found: first run?", path)
		return t
	} else if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(b, &t); err != nil {
		panic(err)
	}
	return t
}

type Channel struct {
	Title         string    `xml:"title"`
	Description   string    `xml:"description"`
	Link          string    `xml:"link"`
	Copyright     string    `xml:"copyright"`
	TTL           int       `xml:"ttl,omitempty"`
	LastBuildDate time.Time `xml:"last_build_date"`
	PubDate       time.Time `xml:"pub_date"`
}
type Item struct {
	Title   string    `xml:"title"`
	Link    string    `xml:"link"`
	GUID    uuid.UUID `xml:"guid"`
	PubDate time.Time `xml:"pub_date"`
}

const initialpublish = "2023-03-14T20:02:03.766615+00:00"

var base = Channel{
	Title:         "efron's blog",
	Description:   "efron's blog about programming w/ a focus on performance",
	Link:          "https://eblog.fly.dev",
	Copyright:     "2023 eblog.fly.dev. all rights reserved",
	LastBuildDate: time.Now(),
	PubDate:       must(time.Parse(time.RFC3339, initialpublish)),
	TTL:           1800,
}

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}
