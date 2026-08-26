# forge

A fast, concurrent static site generator in Go. A site is a folder - a `site.yaml`, a `content/` tree of Markdown, and a theme - and forge renders it to clean-URL HTML with incremental rebuilds that only touch what changed. Single binary, no runtime dependencies

## Why forge

- **Incremental by design** - every buildable thing is a hashed node, so a rebuild only re-renders what actually changed. Edit one post and only that post and its listing regenerate; a warm no-op rebuild is near-instant.
- **Fast** - concurrent builds, flat output, and a content-addressed cache. In local benchmarks against a comparable Go SSG on a 1,000-page corpus it comes out roughly 1.9× faster cold.
- **Clean URLs everywhere** - `/blog/my-post` with no `.html`, portable across GitHub Pages, Netlify, and Cloudflare Pages with zero config.
- **Batteries where they count** - GFM Markdown, server-side syntax highlighting, shortcodes, collections with auto-generated listings, an optional responsive image pipeline, and a `sitemap.xml`.
- **Generic data model** - any key in `site.yaml` or a page's front matter flows straight to your templates as `.Site.<key>` / `.Frontmatter.<key>`. forge reads only the handful of keys it needs and carries the rest through untouched.

## Quick start

A forge site is just a folder. The smallest one that builds:

```
my-site/
  site.yaml
  content/
    home.md
  themes/
    minimal/
      layouts/
        page.html
        listing.html
```

`site.yaml`:

```yaml
title: My Site
theme: minimal
```

`themes/minimal/layouts/page.html` - each layout wraps its content in `{{ define "<name>" }}...{{ end }}`, and a page template emits a full HTML document:

```html
{{ define "page" }}
<!doctype html>
<html lang="en">
  <head><meta charset="utf-8"><title>{{ .Frontmatter.title }}</title></head>
  <body>{{ .Content }}</body>
</html>
{{ end }}
```

Then:

```sh
forge serve .    # dev server at http://localhost:3000, live rebuild on save
forge build .    # one-off build into ./dist
```

Publish the generated `dist/` to any static host.

## CLI

```
forge build <site-dir>    build the site into <site-dir>/dist
forge serve <site-dir>    dev server with live rebuild on change
```

| Flag | Applies to | Meaning |
| --- | --- | --- |
| `--port <n>` | `serve` | Port to listen on. Default `3000`. |
| `--timing` | both | Print a per-phase timing breakdown after building. |

## Project structure

```
my-site/
  site.yaml                 site config
  content/                  your Markdown + assets
    home.md                 ->  /
    about.md                ->  /about
    blog/                   a collection
      index.md              custom index at /blog/   (optional)
      a-post.md             ->  /blog/a-post
    assets/                 copied verbatim to dist/assets/
  layouts/                  optional site-level template overrides
  themes/
    my-theme/
      layouts/              *.html templates
        partials/           snippets: head, nav, footer
        shortcodes/         *.html shortcode templates
      static/               copied verbatim to dist/static/
  dist/                     build output (generated)
  .forge-manifest.json      incremental cache (generated; gitignore it)
```

## Documentation

The full guide - content and front matter, themes and templates, shortcodes, configuration, the image pipeline, and deployment - lives in [`docs/`](docs/), which is itself a forge site or read the sources under `docs/content/`.

The docs also cover [AI theme generation](docs/content/ai-theme-generation.md): a paste-in prompt that turns any capable LLM into a forge theme generator, alongside an [`llms.txt`](docs/llms.txt) that gives models an accurate summary of the whole contract.

## How it works

forge builds a dependency graph of hashed nodes - pages, listings, images, assets, config, theme - where each node declares what it needs via a `Need` call. A node rebuilds only if its own hash changed or one of its inputs is dirty, and results are cached in a manifest between builds. Adding a feature means registering a new node kind, not editing the build loop. See `internal/engine` for the core.

## Building from source

```sh
git clone https://github.com/notblankz/forge
cd forge
go build -o forge .
```

Requires a recent Go toolchain (see `go.mod`). Run the tests with `go test ./...`.

## License

MIT - see [LICENSE](LICENSE).
