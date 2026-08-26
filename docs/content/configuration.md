---
title: Configuration
---

# Configuration

Everything lives in `site.yaml` at your site root. **Every key you put here is available to templates as `.Site.<key>`** - forge doesn't restrict what you can configure. A handful of keys are read by forge itself; everything else is yours to define and render in your theme.

```yaml
title: My Site
theme: my-theme                    # forge reads this - which theme to load
navbar_logo: /assets/logo.svg      # theme data - your theme decides what to do with it

syntax_highlighting: true          # forge reads this - default true
image_sizes: [1600, 1200, 800]     # forge reads this - enables the image pipeline
base_url: https://example.com      # forge reads this - enables sitemap.xml + absolute URLs

# any structure you like; read it in the theme as .Site.nav
nav:
  - label: Blog
    url: /blog/
social:
  - label: GitHub
    url: https://github.com/you
```

## Keys forge reads

Only these keys affect the build:

| Key | Type | Effect |
| --- | --- | --- |
| `theme` | string | Which theme to load. Bare name resolves to `themes/<name>`; a relative or absolute path is used directly. |
| `syntax_highlighting` | bool | Server-side code highlighting. Default `true`. |
| `image_sizes` | list of int | Responsive widths. Non-empty turns the [image pipeline](/images) on; empty or omitted turns it off. |
| `base_url` | string | Your site's public root URL. When set, forge emits a [`sitemap.xml`](/reference); it's also available to templates as `.Site.base_url` for absolute links (e.g. `rel="canonical"`). A trailing slash is optional - forge trims it. |

## Everything else is yours

`title`, `navbar_logo`, `nav`, `social`, or any key you invent - forge carries it through untouched and the theme reads it as `.Site.<key>`. There's no built-in notion of "navigation": `nav` is just a list your theme happens to iterate. A list of maps becomes `range`-able, and each item's keys are read directly:

```html
{{ range .Site.nav }}<a href="{{ .url }}">{{ .label }}</a>{{ end }}
```

Want `nav` items shaped `{name, link, icon}` instead? Write that in `site.yaml` and read `.name`/`.link`/`.icon`. forge doesn't care.

> **Keys are lowercase.** You read `.Site.title`, never `.Site.Title` - the accessor matches the YAML key exactly. This is the single most common thing to silently render blank if you're coming from an older typed config.

> **A missing key is empty, not an error.** `.Site.analytics_id` with no such key renders nothing, and `{{ range .Site.nav }}` over an absent `nav` iterates zero times. Guard optional values with `{{ if .Site.x }}` or `{{ with .Site.x }}`.
