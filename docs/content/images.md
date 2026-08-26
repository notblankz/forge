---
title: Image optimization
---

# Image optimization

Set `image_sizes` in `site.yaml` to enable the responsive image pipeline; omit it (or leave it empty) and images pass through untouched.

```yaml
image_sizes: [1600, 1200, 800, 400]
```

When enabled, calling `image` in a shortcode decodes the source JPEG once and produces a ladder of downscaled, re-compressed variants (never upscaling past the source). It returns:

| Field | Use |
| --- | --- |
| `.Src` | Fallback URL (largest variant) for the `src` attribute. |
| `.Srcset` | The full `srcset` string. |
| `.Width`, `.Height` | Intrinsic size of the fallback - set these on the `<img>` to prevent layout shift. |

```html
{{ with image .src }}
<img src="{{ .Src }}" alt="{{ $.title }}"
     {{ if .Srcset }}srcset="{{ .Srcset }}" sizes="100vw"{{ end }}
     {{ if .Width }}width="{{ .Width }}" height="{{ .Height }}"{{ end }}>
{{ end }}
```

> Guard the attributes with `{{ if .Srcset }}` / `{{ if .Width }}` as shown, and pair them with `img { width: 100%; height: auto; }` in your CSS - the `height` attribute is a presentational hint that will otherwise fix the image's height. The same template then works whether optimization is on or off.

Variants are content-addressed and cached: they regenerate only when the source photo or the size ladder changes. The pipeline currently handles JPEG sources.
