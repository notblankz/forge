package site

import "github.com/notblankz/forge/internal/engine"

func registerNodes(paths SitePaths, themeDir string, listingMembers map[string][]string) {
	engine.Register("@config", configNode{path: paths.Config})
	engine.Register("@theme", themeNode{themeDir: themeDir, siteLayouts: paths.Layouts, contentDir: paths.Content})
	engine.Register("@dir", dirNode{contentDir: paths.Content})
	engine.Register("@page", pageNode{contentDir: paths.Content, destDir: paths.Dest})
	engine.Register("@listing", listingNode{members: listingMembers, destDir: paths.Dest})
	engine.Register("@assets", assetsNode{contentDir: paths.Content, destDir: paths.Dest})
	engine.Register("@theme-static", themeStaticNode{themeDir: themeDir, destDir: paths.Dest})
}
