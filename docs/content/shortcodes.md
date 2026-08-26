---
title: Shortcodes
---

# Shortcodes

Shortcodes are theme-defined snippets you invoke from markdown. Self-closing:

```
{{< hero-photo src="assets/hero.jpg" title="Sunrise" >}}
```

Or paired, wrapping a body that is rendered as markdown and handed to the template as `.Body`:

```
{{< note >}}
This body becomes **.Body** inside note.html.
{{< /note >}}
```

You write one at `themes/<name>/layouts/shortcodes/<name>.html`. Parameters arrive as `.paramName`. Values can be quoted strings, `[bracketed, arrays]`, or bare tokens.

```html
<!-- shortcodes/note.html -->
<aside class="note">{{ .Body }}</aside>
```

## Built-in helpers

Two functions are available inside shortcode templates:

- `readDir "assets/gallery"` - returns the `/assets/…` URLs of every file in that folder, *and* records the folder as a dependency, so adding or changing a file there rebuilds the page.
- `image "assets/hero.jpg"` - returns an optimized image descriptor (see [image optimization](/images)). A no-op passthrough when the image pipeline is off.

```html
<!-- shortcodes/photo-grid.html -->
{{ range readDir .src }}
  {{ with image . }}<img src="{{ .Src }}" srcset="{{ .Srcset }}">{{ end }}
{{ end }}
```

> Shortcode tags inside fenced or inline code are left untouched, so you can document shortcodes in your own pages without them being expanded.
