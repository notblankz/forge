---
title: Syntax highlighting
---

# Syntax highlighting

With `syntax_highlighting: true` (the default), fenced code blocks are highlighted server-side using CSS classes - no client-side JavaScript.

```yaml
syntax_highlighting: true
```

Ship a Chroma stylesheet in your theme's `static/` (for example `chroma.css`) and link it from your `head` partial:

```html
<link rel="stylesheet" href="/static/chroma.css">
```

Because highlighting is class-based, the CSS is what controls the colours - swap the stylesheet to change themes, with no rebuild of your content required.

Set the flag to `false` to emit plain `<pre><code>` and highlight on the client instead (e.g. with highlight.js), or if you simply want unstyled code.
