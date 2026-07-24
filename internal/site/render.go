package site

import (
	"runtime"

	"golang.org/x/sync/errgroup"
)

// renderPages renders and writes all pages concurrently, using a worker pool
// bounded to the number of CPUs. It returns the first error encountered and
// also returns a map of all assets read and used by the page while being rendered
// with the page path as the key
func (b *Builder) renderPages(pages []Page) (map[string][]string, error) {
	g := new(errgroup.Group)
	g.SetLimit(runtime.NumCPU())

	deps := make([][]string, len(pages))
	for i, page := range pages {
		g.Go(func() error {
			html, d, err := b.renderPage(page)
			if err != nil {
				return err
			}
			// store the current page dependency in an indexed slice (concurrent safe)
			deps[i] = d
			return page.write(html)
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	out := make(map[string][]string, len(pages))
	for i, page := range pages {
		out[page.Path] = deps[i]
	}

	return out, nil
}
