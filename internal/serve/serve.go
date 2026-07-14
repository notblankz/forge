package serve

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/notblankz/forge/internal/site"
)

type Config struct {
	site.BuildOptions
	Port int
}

// Run builds the site once, serves the output directory over HTTP on
// localhost:3000, and watches the content and theme directories, rebuilding
// on change until interrupted
func Start(opts Config) error {
	if err := site.Build(opts.BuildOptions); err != nil {
		return err
	}

	go func() {
		fs := http.FileServer(http.Dir(opts.DestRoot))
		http.Handle("/", fs)
		fmt.Printf("\n  forge dev server\n")
		fmt.Printf("  -> local:    http://localhost:%d\n", opts.Port)
		fmt.Printf("  -> watching: %s, themes/\n\n", opts.ContentRoot)
		addr := fmt.Sprintf(":%d", opts.Port)
		if err := http.ListenAndServe(addr, nil); err != nil {
			fmt.Fprintln(os.Stderr, "server error:", err)
		}
	}()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if err := watchDirs(watcher, opts.ContentRoot, "themes"); err != nil {
		return err
	}

	var debounce *time.Timer
	for {
		select {
		case event := <-watcher.Events:
			changed := event.Name
			if debounce != nil {
				debounce.Stop()
			}
			debounce = time.AfterFunc(100*time.Millisecond, func() {
				fmt.Println("changed:", changed, "rebuilding...")
				rebuild(opts.BuildOptions)
			})
		case err := <-watcher.Errors:
			fmt.Fprintln(os.Stderr, "watch error:", err)
		}
	}
}

// watchDirs recursively adds every directory under each root to the watcher,
// since fsnotify does not watch subdirectories automatically
func watchDirs(watcher *fsnotify.Watcher, roots ...string) error {
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return watcher.Add(path)
			}

			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// rebuild regenerates the site, printing the duration on success. Errors are
// reported but not fatal, so a bad save does not stop the dev server
func rebuild(opts site.BuildOptions) {
	start := time.Now()
	if err := site.Build(opts); err != nil {
		fmt.Fprintln(os.Stderr, "rebuild failed:", err)
		return
	}
	fmt.Printf("rebuilt in %s\n", time.Since(start))
}
