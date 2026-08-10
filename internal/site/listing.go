package site

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/notblankz/forge/internal/engine"
)

// listing node
type listingNode struct {
	members map[string][]string
	destDir string
}

func (l listingNode) Hash(key string) (string, error) {
	members := []string{}
	members = append(members, l.members[engine.NodeID(key)]...)
	sort.Strings(members)
	return engine.HashBytes([]byte(strings.Join(members, "\n"))), nil
}

func (l listingNode) Build(ctx *engine.BuildCtx, key string) (engine.Result, error) {
	name := engine.NodeID(key)

	// get the config + theme data using Need
	cfgRes, err := ctx.Need(key, "@config")
	if err != nil {
		return engine.Result{}, err
	}
	cfg := cfgRes.Data.(configData)

	themeRes, err := ctx.Need(key, "@theme")
	if err != nil {
		return engine.Result{}, err
	}
	th := themeRes.Data.(themeData)

	// members: record the dependency edge + read each page's display metadata
	var pages []PageMeta
	for _, path := range l.members[name] {
		pageRes, err := ctx.Need(key, "@page:"+path)
		if err != nil {
			return engine.Result{}, err
		}
		var pm PageMeta
		if err := json.Unmarshal(pageRes.Meta, &pm); err != nil {
			return engine.Result{}, err
		}
		pages = append(pages, pm)
	}

	type listingView struct {
		CommonView
		Name  string
		Pages []PageMeta
	}

	view := listingView{
		CommonView: CommonView{
			Site:      cfg.config,
			PageTitle: name,
		},
		Name:  name,
		Pages: pages,
	}

	var buf bytes.Buffer
	if err := th.theme.ExecuteTemplate(&buf, "listing", view); err != nil {
		return engine.Result{}, fmt.Errorf("listing %q: %w", name, err)
	}

	// write the generated listing html file data to destRoot/<name>/index.html
	outPath := filepath.Join(l.destDir, name, "index.html")
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return engine.Result{}, err
	}

	if err := os.WriteFile(outPath, buf.Bytes(), 0644); err != nil {
		return engine.Result{}, err
	}

	return engine.Result{Outputs: []string{outPath}}, nil

}
