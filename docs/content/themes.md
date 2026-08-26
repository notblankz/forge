---
title: Themes & templates
---

# Themes & templates

A theme lives at `themes/<name>/`:

- `layouts/*.html` - page templates, each wrapped in `{{ define "<name>" }}…{{ end }}`
- `layouts/partials/*.html` - snippets pulled in with `{{ template "head" . }}`
- `layouts/shortcodes/*.html` - [shortcode](/shortcodes) templates
- `static/` - copied verbatim to `/static/`

Templates use Go's `html/template`. A `layouts/` folder at your *site* root overrides theme templates of the same name.

## Which template renders a page

First match wins:

1. the page's front-matter `template`, if it names an existing template;
2. a template matching the file's basename (`about.md` -> an `about` template, if you define one);
3. the generic `page` template.

Collection listings always use `listing`. A `home.md` uses a `home` template if you define one, otherwise `page`.

## What each template receives

This is the contract - know it exactly.

### Page templates - `page`, `home`, filename-matched

| Accessor | What it is |
| --- | --- |
| `.Site` | your whole `site.yaml`, read as `.Site.<key>` (`.Site.title`, `.Site.nav`, …) |
| `.Frontmatter` | the page's whole front matter, read as `.Frontmatter.<key>` (`.Frontmatter.title`, `.Frontmatter.date`, anything) |
| `.URL` | the page's clean URL |
| `.Content` | the rendered HTML body - output with `{{ .Content }}` |

### Listing template - `listing`

| Accessor | What it is |
| --- | --- |
| `.Site` | the same site config |
| `.Name` | the collection's name (its folder) |
| `.Pages` | the entries - `range` it; each is flat with `.title`, `.date`, `.url`, plus any `metadata`-opted-in fields |

> **`.Site` and `.Frontmatter` keys are exactly what you wrote - lowercase.** `.Site.Title` / `.Frontmatter.Title` (capitalized) won't find your `title:` key and render blank.

### Dates in templates

`.Frontmatter.date` (on a page) and `.date` (in a listing) are real time values: format with `{{ .date.Format "02 Jan 2006" }}`, guard empties with `{{ if not .date.IsZero }}`. Write dates unquoted in front matter so YAML keeps them as times rather than strings.

## Site-level overrides

A `layouts/` folder at your site root overrides theme templates of the same name - tweak one page without forking the whole theme.
