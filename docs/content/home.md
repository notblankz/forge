---
title: Overview
---

# forge

A fast, concurrent static site generator in Go. This is the complete guide to building a site and writing a theme.

A forge site is a folder with three things: a `site.yaml` config, a `content/` directory of markdown, and a theme. forge renders every markdown file to a clean-URL HTML page, generates listing pages for your collections, copies your assets, and writes everything to `dist/`.

Builds are **incremental**: forge fingerprints every input and rebuilds only what changed since the last build, caching the rest. It ships as a single binary with no runtime dependencies.

## Where to next

- [Getting started](/getting-started) - install forge and build your first site.
- [Configuration](/configuration) - the `site.yaml` reference.
- [Content & front matter](/content) - what goes in a page, and how any field reaches your templates.
- [Themes & templates](/themes) - write a theme and know exactly what data each template receives.
