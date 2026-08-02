package site

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/notblankz/forge/internal/timing"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"golang.org/x/sync/errgroup"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

type BuildOptions struct {
	SiteRoot string
	DestDir  string
	Timing   bool
}

type Builder struct {
	siteRoot   string
	contentDir string
	themeDir   string
	destDir    string
	config     SiteConfig
	theme      *template.Template
	shortcodes *Shortcodes
	markdown   goldmark.Markdown
}

// Build compiles the site from the content directory: it loads pages, renders
// them concurrManifestently, generates collection listings, and copies assets into the
// output directory
func Build(opts BuildOptions) error {
	t := timing.NewTimer()
	start := time.Now()

	b, err := newBuilder(opts)
	if err != nil {
		return err
	}

	t.Mark("newBuilder")

	paths, err := collectContent(b.contentDir)
	if err != nil {
		return err
	}

	t.Mark("collect content")

	pages := make([]Page, len(paths))
	g := new(errgroup.Group)
	g.SetLimit(runtime.NumCPU())
	for i, path := range paths {
		g.Go(func() error {
			page, err := b.loadPage(path)
			if err != nil {
				return err
			}
			pages[i] = page
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	t.Mark("load + hash pages")

	// Create the collections map for the different collections in content/
	collections, err := groupCollections(pages, b.contentDir)
	if err != nil {
		return err
	}

	t.Mark("group collections")

	// Fingerprint this build, and load the last one to compare against
	prevManifest, err := b.loadManifest()
	if err != nil {
		return err
	}
	currManifest, err := b.buildManifestMap(pages, collections, prevManifest)
	if err != nil {
		return err
	}

	t.Mark("build manifest")

	// Decide what to rebuild
	var dirtyPages []Page
	var dirtyCollections []*Collection

	rebuildAll := prevManifest == nil

	var changed map[string]struct{}
	if !rebuildAll {
		// Get the dirty set of pages + collections
		changed = diffManifests(prevManifest, currManifest)
		_, configChanged := changed["@config"]
		_, themeChanged := changed["@theme"]
		rebuildAll = configChanged || themeChanged
	}

	if rebuildAll {
		// cold build, or a universal dep (@config/@theme) changed -> full rebuild
		dirtyPages = pages
		for _, c := range collections {
			dirtyCollections = append(dirtyCollections, c)
		}
	} else {
		// We only rebuild the dirty pages and collections by comparing
		// the previous manifest

		// build the DAG graph
		g, err := buildGraphFromManifest(prevManifest, currManifest)
		if err != nil {
			return err
		}

		depChanged, err := b.pagesWithChangedDeps(prevManifest)
		if err != nil {
			return err
		}
		for id := range depChanged {
			changed[id] = struct{}{}
		}
		dirty := g.Dirty(changed)

		// Go through the full pages and collections slice, add only
		// those pages and collection which also exists in dirty Set
		for _, p := range pages {
			if _, ok := dirty[p.Path]; ok {
				dirtyPages = append(dirtyPages, p)
			}
		}
		for _, c := range collections {
			if _, ok := dirty["@listing:"+c.Name]; ok {
				dirtyCollections = append(dirtyCollections, c)
			}
		}
	}

	t.Mark("diff + dirty")

	// Render all the standalone pages
	renderedDeps, err := b.renderPages(dirtyPages)
	if err != nil {
		return err
	}

	t.Mark("render pages")

	for _, c := range dirtyCollections {
		if c.Index != nil {
			continue // has index.md hence use that, renders via normal page path
		}
		if err := b.generateListingPage(c); err != nil {
			return err
		}
	}

	t.Mark("generate listings")

	assetDests, err := b.copyAssets()
	if err != nil {
		return err
	}
	currManifest["@assets"] = manifestEntry{Outputs: assetDests}

	themeDests, err := b.copyThemeAssets()
	if err != nil {
		return err
	}
	currManifest["@theme-static"] = manifestEntry{Outputs: themeDests}

	t.Mark("copy assets")

	// Delete only what the last build wrote that this build no longer produces
	if err := b.cleanOrphans(prevManifest, currManifest); err != nil {
		return err
	}

	if err := b.recordRenderedDeps(currManifest, renderedDeps); err != nil {
		return err
	}

	t.Mark("clean orphans")

	// we write the new manifest at the end to preserve
	// the old manifest in case of an error mid-build
	if err := b.saveManifest(currManifest); err != nil {
		return err
	}
	t.Mark("save manifest")

	fmt.Printf("built in %s\n", time.Since(start))

	if opts.Timing {
		t.Report(os.Stdout)
	}

	return nil
}

// newBuilder constructs a Builder from the given options, deriving the site
// layout and loading the config and theme the rest of the build depends on
func newBuilder(opts BuildOptions) (*Builder, error) {
	paths := NewSitePaths(opts.SiteRoot, opts.DestDir)

	config, err := LoadConfig(paths.Config)
	if err != nil {
		return nil, err
	}

	extensions := []goldmark.Extender{extension.GFM}

	if config.SyntaxHighlighting {
		extensions = append(extensions, highlighting.NewHighlighting(
			highlighting.WithFormatOptions(
				chromahtml.WithClasses(true),
			),
		))
	}

	markdown := goldmark.New(goldmark.WithExtensions(extensions...))

	themeDir := ResolveThemeDir(paths.Root, config.Theme)

	theme, err := loadTheme(themeDir, paths.Layouts)
	if err != nil {
		return nil, err
	}

	shortcodes, err := loadShortcodes(themeDir, paths.Layouts, paths.Content, markdown)
	if err != nil {
		return nil, err
	}

	return &Builder{
		siteRoot:   paths.Root,
		contentDir: paths.Content,
		themeDir:   themeDir,
		destDir:    paths.Dest,
		config:     config,
		theme:      theme,
		shortcodes: shortcodes,
		markdown:   markdown,
	}, nil
}

// collectContent walks the content root recursively and returns the paths of
// all markdown (.md) files found in a slice
func collectContent(root string) ([]string, error) {
	res := make([]string, 0)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {

		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		ext := filepath.Ext(d.Name())
		if ext == ".md" {
			res = append(res, path)
		}

		return nil
	})

	return res, err
}
