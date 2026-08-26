---
title: Builds, CLI & deploy
---

# Builds, CLI & deploy

The biggest reason forge stays fast is due to it's incremental builds using DAGs and caching the output in `.forge-manifest.json` that allows `forge` to not have to build the entire site again and instead just build the pages that have been dirtied since the last build

## Incremental builds

forge records a `.forge-manifest.json` holding each node's hash, inputs, and outputs. On the next build it re-fingerprints your sources, computes the dirty set, and rebuilds only that - the rest is served from cache.

Editing one post re-renders that post and its collection listing and nothing else; a warm rebuild with no changes is near-instant. The manifest is safe to add to `.gitignore`.

Run any build with `--timing` to see the per-phase breakdown:

```sh
forge build . --timing
```

## CLI

```
forge build <site-dir>      build the site into <site-dir>/dist
forge serve <site-dir>      dev server with live rebuild on change
```

| Flag | Applies to | Meaning |
| --- | --- | --- |
| `--port <n>` | `serve` | Port to listen on. Default `3000`. |
| `--timing` | both | Print a per-phase timing breakdown after building. |

## Sitemap

Set [`base_url`](/configuration) in `site.yaml` and forge writes a `sitemap.xml` into `dist/`, listing every page and auto-generated listing as an absolute URL. Pages that carry a `date` in their front matter also get a `<lastmod>`:

```yaml
base_url: https://example.com
```

```xml
<url>
  <loc>https://example.com/blog/hello</loc>
  <lastmod>2026-07-24</lastmod>
</url>
```

Without `base_url` no sitemap is written - a sitemap of relative URLs is useless. It regenerates whenever a page is added or removed, a page's date changes, or `base_url` changes; remove `base_url` later and the next build cleans up the stale `sitemap.xml` for you.

## Deploying

Run `forge build .` and publish the generated `dist/` folder to any static host. Clean URLs work as-is on GitHub Pages, Netlify, and Cloudflare Pages - no redirect rules or extra configuration required.

These very docs are a forge site: the `docs/` folder builds with `forge build docs`, and the result is what you're reading.
