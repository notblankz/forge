---
title: Content & front matter
---

# Content & front matter

Every `.md` file under `content/` becomes a page. Front matter is optional YAML between `---` fences, and **any key you write is available to the page template as `.Frontmatter.<key>`** - forge doesn't restrict the fields.

```markdown
---
title: My Project
date: 2026-01-15
description: A short summary.
github: https://github.com/you/thing
tech: [Go, HTML, CSS]
featured: true
---
# body starts here
```

A page template reads them flat:

```html
<h1>{{ .Frontmatter.title }}</h1>
<a href="{{ .Frontmatter.github }}">Source</a>
{{ range .Frontmatter.tech }}<span>{{ . }}</span>{{ end }}
{{ if .Frontmatter.featured }}★{{ end }}
```

## Nothing is required

There are no mandatory fields - a file with no front matter still builds, rendering just its body, and forge never fails a build over missing front matter. `date` and `description` are conventions your theme leans on, not rules forge enforces. A standalone page (a home page, an about page) can skip all of it.

`title` is the one field forge fills in for you: when a page omits it, forge derives a `title` from the filename (`my-first-post.md` becomes `my first post`), so both the page's own `.Frontmatter.title` and its [listing](/collections) entry always have one. Set `title` yourself whenever you want something other than the filename. `date` stays optional and only appears in a listing when set. Guard optional fields with `{{ if .Frontmatter.x }}`.

> **Front matter is accessed as `.Frontmatter.<key>`, lowercase.** Coming from typed front matter, `.Frontmatter.Title` and a standalone `.PageTitle` no longer exist - it's `.Frontmatter.title`, using the exact key you wrote.

## Types

Values keep their YAML types on the page: strings, booleans (`{{ if .Frontmatter.featured }}`), lists (`{{ range .Frontmatter.tech }}`), and nested maps (`.Frontmatter.author.name`).

**Dates are the one to watch.** Write the date *unquoted* - `date: 2026-01-15` - and YAML resolves it to a real time value, so `{{ .Frontmatter.date.Format "02 Jan 2006" }}` works. Quote it and you get a plain string with no `.Format`. Guard empty dates with `{{ if not .Frontmatter.date.IsZero }}`.

Markdown is GitHub-Flavored: tables, task lists, strikethrough, and autolinks all work.

## URLs

forge writes flat files with extensionless clean URLs. Directory indexes (`home.md`, `index.md`) keep `index.html` so folder URLs resolve everywhere.

| Source | Output file | URL |
| --- | --- | --- |
| `content/home.md` | `dist/index.html` | `/` |
| `content/about.md` | `dist/about.html` | `/about` |
| `content/blog/a-post.md` | `dist/blog/a-post.html` | `/blog/a-post` |
| `content/blog/index.md` | `dist/blog/index.html` | `/blog/` |

The page's own URL is available in its template as `.URL`. These resolve on GitHub Pages, Netlify, and Cloudflare Pages with no config, and `forge serve` resolves them locally too.
