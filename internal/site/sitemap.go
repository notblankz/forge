// Credit to Claude Opus 4.8 for this file
package site

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/notblankz/forge/internal/engine"
)

// @sitemap - emits dist/sitemap.xml listing every page and auto-listing URL as
// absolute links. Only scheduled when base_url is set (a sitemap with relative
// URLs is useless). It enumerates its members up front, like @listing, rather
// than discovering them during the build
type sitemapNode struct {
	baseURL  string
	pages    []string // content paths, one per @page
	listings []string // auto-listing names
	destDir  string
	hash     string // fingerprint, folded once at construction
}

// newSitemapNode owns its inputs by copying them, sorts the copies for a
// canonical order, and folds the fingerprint once. Hash and Build then just read
// the stored results - no per-call sort, no mutation of the caller's slices, and
// no ordering dependency between Hash and Build
func newSitemapNode(baseURL string, pages, listings []string, destDir string) sitemapNode {
	pages = append([]string(nil), pages...)
	listings = append([]string(nil), listings...)
	sort.Strings(pages)
	sort.Strings(listings)

	var b strings.Builder
	b.WriteString(baseURL + "\n")
	b.WriteString(strings.Join(pages, "\n") + "\n")
	b.WriteString(strings.Join(listings, "\n"))

	return sitemapNode{
		baseURL:  baseURL,
		pages:    pages,
		listings: listings,
		destDir:  destDir,
		hash:     engine.HashBytes([]byte(b.String())),
	}
}

// Hash returns the fingerprint folded at construction from base_url and the
// sorted member sets: a base_url edit or any page/listing add/remove changes it
// and rebuilds the sitemap. Page content edits (e.g. a changed date) propagate
// through the @page edges Need'ed in Build
func (s sitemapNode) Hash(string) (string, error) {
	return s.hash, nil
}

// sitemap XML shape per https://www.sitemaps.org/protocol.html. encoding/xml
// escapes reserved characters (&, <) in the URLs for us
type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

const sitemapNS = "http://www.sitemaps.org/schemas/sitemap/0.9"

func (s sitemapNode) Build(ctx *engine.BuildCtx, key string) (engine.Result, error) {
	set := sitemapURLSet{Xmlns: sitemapNS}

	// pages: Need each one for its display metadata (url + date). The edge also
	// means a page content change dirties the sitemap, so lastmod stays accurate
	for _, path := range s.pages {
		res, err := ctx.Need(key, "@page:"+path)
		if err != nil {
			return engine.Result{}, err
		}

		var pm PageMetadata
		if err := json.Unmarshal(res.Meta, &pm); err != nil {
			return engine.Result{}, err
		}

		url, _ := pm["url"].(string)
		entry := sitemapURL{Loc: s.baseURL + url}
		if d, ok := pm["date"].(time.Time); ok && !d.IsZero() {
			entry.LastMod = d.Format("2006-01-02")
		}
		set.URLs = append(set.URLs, entry)
	}

	// listings: the URL is derivable from the name, so there's no metadata to read
	for _, name := range s.listings {
		set.URLs = append(set.URLs, sitemapURL{Loc: s.baseURL + "/" + name + "/"})
	}

	body, err := xml.MarshalIndent(set, "", "  ")
	if err != nil {
		return engine.Result{}, err
	}
	out := append([]byte(xml.Header), body...)

	if err := os.MkdirAll(s.destDir, 0755); err != nil {
		return engine.Result{}, err
	}

	outPath := filepath.Join(s.destDir, "sitemap.xml")
	if err := os.WriteFile(outPath, out, 0644); err != nil {
		return engine.Result{}, err
	}

	return engine.Result{Outputs: []string{outPath}}, nil
}
