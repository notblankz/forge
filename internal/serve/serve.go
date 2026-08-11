package serve

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/notblankz/forge/internal/site"
)

type Config struct {
	site.BuildOptions
	Port int
}

// Start builds the site once, serves the output directory over HTTP, and
// watches the content, site-level layouts, and themes directories, rebuilding
// on change until interrupted
func Start(opts Config) error {
	if err := site.Build(opts.BuildOptions); err != nil {
		return err
	}

	paths := site.NewSitePaths(opts.SiteRoot, opts.DestDir)

	go func() {
		fileServer := http.FileServer(http.Dir(paths.Dest))
		http.Handle("/", cleanURLHandler(paths.Dest, fileServer))
		fmt.Printf("\n  forge dev server\n")
		fmt.Printf("  -> local:    http://localhost:%d\n", opts.Port)
		fmt.Printf("  -> watching: %s, %s, %s\n\n", paths.Content, paths.Layouts, site.ThemesRoot)
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

	config, err := site.LoadConfig(paths.Config)
	if err != nil {
		return err
	}

	themeDir := site.ResolveThemeDir(paths.Root, config.Theme)

	if err := watchDirs(watcher, paths.Content, paths.Layouts, themeDir); err != nil {
		return err
	}
	watcher.Add(filepath.Join(paths.Root, "site.toml"))

	var debounce *time.Timer
	for {
		select {
		case event := <-watcher.Events:
			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if err := watchDirs(watcher, event.Name); err != nil {
						fmt.Fprintln(os.Stderr, "watch error:", err)
					}
				}
			}
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
		if _, err := os.Stat(root); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}

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

// cleanURLHandler serves extensionless clean URLs (/blog/post) from flat files
// (blog/post.html). Directory requests (/, /blog/) and real files (assets)
// fall through to the file server untouched
func cleanURLHandler(root string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if path.Ext(p) == "" && !strings.HasSuffix(p, "/") {
			full := filepath.Join(root, filepath.FromSlash(p))
			if _, err := os.Stat(full); err != nil { // not already a real file/dir
				if html := full + ".html"; fileExists(html) {
					http.ServeFile(w, r, html)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func fileExists(name string) bool {
	st, err := os.Stat(name)
	return err == nil && !st.IsDir()
}
