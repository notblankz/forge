---
title: Getting started
---

# Getting started

## Install

With Go installed:

```sh
go install github.com/notblankz/forge@latest
```

Or build from source:

```sh
git clone https://github.com/notblankz/forge
cd forge
go build -o forge .
```

## Quick start

A forge site is just a folder. Here's the smallest one that builds.

### 1 · Config

A `site.yaml` at the site root:

```yaml
title: My Site
theme: minimal
```

### 2 · Content

Markdown under `content/`:

```
content/
  home.md          ->  /
  about.md         ->  /about
  blog/
    first-post.md  ->  /blog/first-post
```

A page is markdown with optional front matter:

```markdown
---
title: Hello
date: 2026-01-01
---
# Hello

This is **markdown**, rendered to a clean URL.
```

### 3 · Theme

Under `themes/<theme name>/layouts/`. Each file *defines* a named template; forge needs at least `page` and `listing`.

`themes/<theme name>/layouts/page.html`:

```html
{{ define "page" }}
<!doctype html>
<html><head><title>{{ .Frontmatter.title }}</title></head>
<body>{{ .Content }}</body>
</html>
{{ end }}
```

`themes/<theme name>/layouts/listing.html`:

```html
{{ define "listing" }}
<!doctype html>
<html><head><title>{{ .Name }}</title></head>
<body>
  <h1>{{ .Name }}</h1>
  <ul>{{ range .Pages }}<li><a href="{{ .url }}">{{ .title }}</a></li>{{ end }}</ul>
</body>
</html>
{{ end }}
```

> A page template reads front matter as `.Frontmatter.<key>`; a listing entry can range through `.Pages` and access its fields directly - `.title`, `.url`, `.date`. The full data each template receives is on [Themes & templates](/themes).

### 4 · Build

```sh
forge serve .     # http://localhost:3000, rebuilds on save
forge build .     # one-off build into ./dist
```

## Project structure

```
my-site/
  site.yaml                 site config
  content/                  your markdown + assets
    home.md                 ->  /
    about.md                ->  /about
    blog/                   a collection
      index.md              custom index at /blog/   (optional)
      a-post.md             ->  /blog/a-post
    assets/                 copied verbatim to /assets/
  layouts/                  optional site-level template overrides
  themes/
    my-theme/
      layouts/              *.html templates
        partials/           snippets: head, nav, footer
        shortcodes/         *.html shortcode templates
      static/               copied verbatim to /static/
  dist/                     build output (generated)
  .forge-manifest.json      incremental cache (generated; gitignore it)
```
