// copyrel copies a directory tree from one location to another.
// (C) Efron Licht, 2023, for educational use for 'rootinit'
package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

var quiet = flag.Bool("q", false, "quiet mode: don't print non-error messages")

// debugf is a logging function that is only enabled when the -q flag is not set.
var debugf = func(format string, args ...interface{}) {}

func main() {
	start := time.Now()
	log.SetPrefix("copyrel\t")
	flag.Parse()
	if !*quiet {
		debugf = log.Printf
	}
	args := flag.Args()
	if len(args) != 2 {
		log.Print("expected two command-line arguments")
		log.Fatal("USAGE: copyrel srcdir dstdir")
	}
	srcDir, dstDir := args[0], args[1]

	srcDir, err := filepath.Abs(srcDir)
	if err != nil {
		log.Fatalf("failed to get absolute path of source directory: %v", err)
	}
	dstDir, err = filepath.Abs(dstDir)
	if err != nil {
		log.Fatalf("failed to get absolute path of destination directory: %v", err)
	}
	debugf("copying %s to %s", srcDir, dstDir)

	if err := os.MkdirAll(dstDir, 0o777); err != nil {
		log.Fatalf("failed to create destination directory: %v", err)
	}
	wg := &sync.WaitGroup{}

	// copied and errs are used to count the number of files copied and the number of errors, respectively.
	// they're atomics since we're incrementing them from multiple goroutines.
	var copied, errs atomic.Int64

	// walkfn is called for each file in the directory tree.
	walkfn := func(srcPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath := srcPath[len(srcDir):]          // relative path of the source file to it's position in the source directory
		dstPath := filepath.Join(dstDir, relPath) // absolute path of the target file in the destination directory
		if d.IsDir() {
			// create the corresponding directory in the destination
			if err := os.MkdirAll(dstPath, 0o777); err != nil {
				log.Printf("failed to create destination directory %q: %v", dstPath, err)
			}
			return nil
		}
		// it's a file, not a directory; copy it in a new goroutine

		wg.Add(1) // increment the waitgroup before starting the goroutine
		go func() {
			defer wg.Done()
			if err := copyFile(dstPath, srcPath); err != nil {
				log.Printf("failed to copy %q to %q: %v", srcPath, dstPath, err)
				errs.Add(1)
			} else {
				debugf(".%s: ok", srcPath[len(srcDir):])
				copied.Add(1)
			}
		}()
		return nil
	}

	if err := filepath.WalkDir(srcDir, walkfn); err != nil {
		log.Fatalf("failed to walk directory tree: %v", err)
	}
	wg.Wait()
	debugf("finished in %.2f ms", time.Since(start).Seconds()*1000)
	nOK, nErr := copied.Load(), errs.Load()
	if nErr == 0 {
		debugf("copied %d files, %d errors", nOK, nErr)
		return
	}
	log.Fatalf("copied %d files, %d errors", nOK, nErr)
}

func copyFile(dstPath, srcPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open %q: %w", srcPath, err)
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create %q: %w", srcPath, err)
	}
	defer dstFile.Close()
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy %q -> %q: %w", srcPath, dstPath, err)
	}
	return nil
}
