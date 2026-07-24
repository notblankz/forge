package site

import (
	"fmt"
	"html/template"
	"io/fs"
	"path/filepath"
	"runtime"
	"time"

	"github.com/notblankz/forge/internal/dag"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"golang.org/x/sync/errgroup"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

type BuildOptions struct {
	SiteRoot string
	DestDir  string
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
// them concurrently, generates collection listings, and copies assets into the
// output directory
func Build(opts BuildOptions) error {
	start := time.Now()

	b, err := newBuilder(opts)
	if err != nil {
		return err
	}

	paths, err := collectContent(b.contentDir)
	if err != nil {
		return err
	}

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

	// Create the collections map for the different collections in content/
	collections, err := groupCollections(pages, b.contentDir)
	if err != nil {
		return err
	}

	// Fingerprint this build, and load the last one to compare against
	prev, err := b.loadManifest()
	if err != nil {
		return err
	}
	curr, err := b.buildManifestMap(pages, prev)
	if err != nil {
		return err
	}

	// Decide what to rebuild
	var dirtyPages []Page
	var dirtyCollections []*Collection

	if prev == nil {
		// Since no previous manifest exists we do a cold rebuild
		dirtyPages = pages
		for _, c := range collections {
			dirtyCollections = append(dirtyCollections, c)
		}
	} else {
		// We only rebuild the dirty pages and collections by comparing
		// the previous manifest

		// build the DAG graph
		g, err := b.buildGraph(pages, collections)
		if err != nil {
			return err
		}
		// Get the dirty set of pages + collections
		changed := diffManifests(prev, curr)
		depChanged, err := b.depChangedPages(prev)
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

		// cleanup output directory
		if err := b.deleteRemovedOutputs(prev, curr); err != nil {
			return err
		}
	}

	// Render all the standalone pages
	renderedDeps, err := b.renderPages(dirtyPages)
	if err != nil {
		return err
	}

	for _, c := range dirtyCollections {
		if c.Index != nil {
			continue // has index.md hence use that, renders via normal page path
		}
		if err := b.generateListingPage(c); err != nil {
			return err
		}
	}

	if err := b.copyAssets(); err != nil {
		return err
	}

	if err := b.copyThemeAssets(); err != nil {
		return err
	}

	if err := b.recordDeps(curr, renderedDeps); err != nil {
		return err
	}

	fmt.Printf("built in %s\n", time.Since(start))

	// we write the new manifest at the end to preserve
	// the old manifest in case of an error mid-build
	return b.saveManifest(curr)
}

// newBuilder constructs a Builder from the given options, deriving the site
// layout and loading the config and theme the rest of the build depends on
func newBuilder(opts BuildOptions) (*Builder, error) {
	paths := NewSitePaths(opts.SiteRoot, opts.DestDir)

	markdown := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(true),
				),
			),
		),
	)

	config, err := loadConfig(paths.Config)
	if err != nil {
		return nil, err
	}

	themeDir := filepath.Join(ThemesRoot, config.Theme)

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

func (b *Builder) buildGraph(pages []Page, collectionsMap map[string]*Collection) (*dag.Graph, error) {

	// create a new graph
	g := dag.NewGraph()

	// add the default node for site.toml and the entire theme directory as a whole
	g.AddNode("@config")
	g.AddNode("@theme")

	// Each page becomes a node and is joined to the site.toml (@config)
	// since if the @config changes every page is dirtied
	for _, page := range pages {
		g.AddNode(page.Path)
		if err := g.AddEdge("@config", page.Path); err != nil {
			return nil, err
		}
		if err := g.AddEdge("@theme", page.Path); err != nil {
			return nil, err
		}
	}

	// A collection's auto-generated listing (@listing:<name>) is its own node,
	// rebuilt whenever any page it lists changes (it shows their titles/dates)
	// Collections with a custom index.md have no auto-listing, so skip them
	for collectionName, collection := range collectionsMap {
		if collection.Index != nil {
			continue
		}
		collectionID := "@listing:" + collectionName
		g.AddNode(collectionID)
		for _, page := range collection.Pages {
			if err := g.AddEdge(page.Path, collectionID); err != nil {
				return nil, err
			}
		}
	}

	// Now we have all the files added to the graph as Nodes

	return g, nil
}
