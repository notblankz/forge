package site

import (
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/notblankz/forge/internal/engine"
	"golang.org/x/image/draw"
)

type imageNode struct {
	contentDir string
	destDir    string
	sizes      []int
}

type imageVariant struct {
	URL   string `json:"url"`
	Width int    `json:"width"`
}

type imageMeta struct {
	Src      string         `json:"src"`
	Srcset   string         `json:"srcset"`
	Width    int            `json:"width"`
	Height   int            `json:"height"`
	Variants []imageVariant `json:"variants"`
}

func (i imageNode) Hash(key string) (string, error) {
	// we use Hash(NodeID|size|mtime) as the source of truth
	fi, err := os.Stat(filepath.Join(i.contentDir, engine.NodeID(key)))
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s|%d|%d\n", engine.NodeID(key), fi.Size(), fi.ModTime().UnixNano())

	return engine.HashBytes([]byte(b.String())), nil
}

func (i imageNode) Build(ctx *engine.BuildCtx, key string) (engine.Result, error) {
	src := engine.NodeID(key)

	if len(i.sizes) == 0 {
		meta, err := json.Marshal(imageMeta{Src: "/" + src})
		if err != nil {
			return engine.Result{}, err
		}
		return engine.Result{Meta: meta}, nil
	}

	f, err := os.Open(filepath.Join(i.contentDir, src))
	if err != nil {
		return engine.Result{}, err
	}
	defer f.Close()

	img, err := jpeg.Decode(f)
	if err != nil {
		return engine.Result{}, fmt.Errorf("image %q: %w", src, err)
	}
	srcBounds := img.Bounds()
	srcW, srcH := srcBounds.Dx(), srcBounds.Dy()

	// variants live in a folder named after the source (minus extension), so
	// a/hero.jpg -> a/hero/1600.jpg and never collide across directories
	rel := strings.TrimSuffix(src, filepath.Ext(src))
	outDir := filepath.Join(i.destDir, rel)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return engine.Result{}, err
	}

	var outputs []string
	var variants []imageVariant

	for _, w := range chooseWidths(i.sizes, srcW) {
		h := srcH * w / srcW
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, srcBounds, draw.Over, nil)

		outPath := filepath.Join(outDir, fmt.Sprintf("%d.jpg", w))
		out, err := os.Create(outPath)
		if err != nil {
			return engine.Result{}, err
		}
		if err := jpeg.Encode(out, dst, &jpeg.Options{Quality: 80}); err != nil {
			out.Close()
			return engine.Result{}, err
		}
		out.Close()

		url := "/" + filepath.ToSlash(filepath.Join(rel, fmt.Sprintf("%d.jpg", w)))
		outputs = append(outputs, outPath)
		variants = append(variants, imageVariant{
			URL:   url,
			Width: w,
		})
	}

	var srcset strings.Builder
	for idx, v := range variants {
		if idx > 0 {
			srcset.WriteString(", ")
		}
		fmt.Fprintf(&srcset, "%s %dw", v.URL, v.Width)
	}

	largest := variants[len(variants)-1]

	meta, err := json.Marshal(imageMeta{
		Src:      largest.URL,
		Srcset:   srcset.String(),
		Width:    largest.Width,
		Height:   srcH * largest.Width / srcW,
		Variants: variants,
	})

	if err != nil {
		return engine.Result{}, err
	}

	return engine.Result{
		Outputs: outputs,
		Meta:    meta,
	}, nil

}

// chooseWidths keeps ladder sizes no larger than the source (never upscale). If
// the source is smaller than the whole ladder, it emits a single native-width
// variant so there's always at least one output.
func chooseWidths(sizes []int, srcW int) []int {
	var ws []int
	for _, s := range sizes {
		if s <= srcW {
			ws = append(ws, s)
		}
	}
	if len(ws) == 0 {
		ws = []int{srcW}
	}
	sort.Ints(ws)
	return ws
}
