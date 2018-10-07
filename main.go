// Copyright 2017 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/tmthrgd/fasttemplate"
)

func newPath(tmpl *fasttemplate.Template, path string) string {
	dir, file := filepath.Split(path)
	ext := filepath.Ext(file)
	name := strings.TrimSuffix(file, ext)

	return tmpl.ExecuteString(map[string]interface{}{
		"path": path,
		"dir":  dir,
		"file": file,
		"name": name,
		"ext":  ext,
	})
}

type workUnit struct {
	path, newPath string
}

func (wrk *workUnit) convert(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "tifig", "-p", "-q", "100", wrk.path, wrk.newPath)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func worker(ctx context.Context, ch <-chan workUnit, wg *sync.WaitGroup) {
	for work := range ch {
		if err := work.convert(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "<%s>: %v\n", work.path, err)
		}

		wg.Done()
	}
}

func fileIsHEIC(path string) bool {
	switch strings.ToUpper(filepath.Ext(path)) {
	case ".HEIC", ".HEIF":
		return true
	default:
		return false
	}
}

func main() {
	outPath := flag.String("out", "{dir}{file}.jpg", "the output path template")
	recurse := flag.Bool("recurse", true, "whether to walk into child directories")
	flag.Parse()

	if _, err := exec.LookPath("tifig"); err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup

	work := make(chan workUnit)
	defer close(work)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < 32; i++ {
		go worker(ctx, work, &wg)
	}

	dir := flag.Arg(0)
	if dir == "" {
		dir = "."
	}

	pathTmpl, err := fasttemplate.NewTemplate(*outPath, "{", "}")
	if err != nil {
		log.Fatal(err)
	}

	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if !*recurse && path != dir {
				return filepath.SkipDir
			}

			return nil
		}

		if !fileIsHEIC(path) {
			return nil
		}

		newPath := newPath(pathTmpl, path)

		if infoOut, err := os.Stat(newPath); err == nil {
			if info.ModTime().Before(infoOut.ModTime()) {
				return nil
			}
		} else if !os.IsNotExist(err) {
			return err
		}

		wg.Add(1)
		work <- workUnit{path, newPath}
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		wg.Wait()
	}()

	// termination handler
	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	select {
	case <-done:
	case <-term:
		signal.Stop(term)

		cancel()
		<-done
	}
}
