---
title: Collections & listings
---

# Collections & listings

Any subfolder of `content/` is a collection. Whether forge auto-generates an index depends on `index.md`:

- **No `index.md`** - forge generates a listing at `/<collection>/` using the `listing` template, passing it every entry's metadata.
- **Has `index.md`** - your hand-written page is used, and no auto-listing is generated.

## What a listing entry contains

A listing template receives `.Pages` - one entry per member. Each entry is flat, and always carries three fields:

| Field | |
| --- | --- |
| `.title` | the page's title - always present; forge derives it from the filename when a page omits `title` |
| `.date` | the page's date as a real time value (`.date.Format`, `.date.IsZero`); zero when the page has no `date`, so guard it |
| `.url` | the page's clean URL, always derived |

> **A listing entry always has a `title`.** A page in a collection doesn't have to set one - forge derives a missing `title` from the filename (`my-first-post.md` becomes `my first post`), so a row always has a usable label (and the page's own `<title>` too). Set `title` in front matter whenever you want something other than the filename. `date` stays optional: a page without one still lists, which is why the example guards it with `{{ if not .date.IsZero }}`.

```html
{{ define "listing" }}
<h1>{{ .Name }}</h1>
<ul>
  {{ range .Pages }}
  <li>
    <a href="{{ .url }}">{{ .title }}</a>
    {{ if not .date.IsZero }}<time>{{ .date.Format "02 Jan 2006" }}</time>{{ end }}
  </li>
  {{ end }}
</ul>
{{ end }}
```

## Showing custom fields - opt in with `metadata`

By default a listing only sees `title`, `date`, and `url`. To surface *other* front-matter fields to a listing - a `featured` star, a `tech` list on a projects grid - list their names in the page's `metadata` field:

```markdown
---
title: forge
date: 2026-01-01
tech: [Go, HTML]
featured: true
metadata: [tech, featured]     # expose these to listings
---
```

Now the listing entry has `.tech` and `.featured` alongside the standard three:

```html
{{ range .Pages }}
  {{ if .featured }}★{{ end }} {{ .title }}
  {{ range .tech }}<span>{{ . }}</span>{{ end }}
{{ end }}
```

> **It's opt-in, and a missing field is silently blank.** If a listing template reads `.tech` but a page didn't put `tech` in its `metadata` list, `.tech` is just empty - no error, no warning. When a custom field "isn't showing up" in a listing, the `metadata` list is the first thing to check. You never list `title`/`date`/`url` - they're always there.

> **Why opt-in?** Listing metadata is persisted in the build cache, so keeping it to the fields a listing actually needs keeps that cache lean. The cost is one line of front matter per page that wants extra listing fields.

## Type notes

Custom fields reach listings through the cache as JSON, which softens two types: a **number** comes back as a float (fine to display, not an int for arithmetic), and a **custom date field** other than the standard `date` comes back as a *string* (no `.Format`). The standard `.date` is re-typed for you and stays a real time value.
