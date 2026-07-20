package site

import (
	"runtime"

	"golang.org/x/sync/errgroup"
)

// renderPages renders and writes all pages concurrently, using a worker pool
// bounded to the number of CPUs. It returns the first error encountered
func (b *Builder) renderPages(pages []Page) error {
	g := new(errgroup.Group)
	g.SetLimit(runtime.NumCPU())

	for _, page := range pages {
		g.Go(func() error {
			html, err := b.renderPage(page)
			if err != nil {
				return err
			}
			return page.write(html)
		})
	}

	return g.Wait()
}
