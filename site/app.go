package app

import (
	"embed"
	"net/http"

	"github.com/river-now/river"
	"github.com/river-now/river/kit/colorlog"
	"github.com/river-now/river/kit/theme"
	"github.com/river-now/river/kit/xyz"
	"github.com/river-now/river/wave"
)

const (
	Domain          = "river.now"
	Origin          = "https://" + Domain
	SiteTitle       = "river.now"
	SiteDescription = "River is a framework for writing modern, type-safe web applications with Go and TypeScript."
)

var River = &river.River{
	Wave: Wave,
	GetDefaultHeadBlocks: func(r *http.Request) ([]*river.HeadBlock, error) {
		root := xyz.GetRootURL(r)
		imgURL := root + Wave.GetPublicURL("river-banner.webp")
		currentURL := root + r.URL.Path

		return []*river.HeadBlock{
			{Tag: "title", InnerHTML: SiteTitle},
			{Tag: "meta", Attributes: map[string]string{"name": "description", "content": SiteDescription}},

			{Tag: "meta", Attributes: map[string]string{"property": "og:title", "content": SiteTitle}},
			{Tag: "meta", Attributes: map[string]string{"property": "og:description", "content": SiteDescription}},
			{Tag: "meta", Attributes: map[string]string{"property": "og:type", "content": "website"}},
			{Tag: "meta", Attributes: map[string]string{"property": "og:image", "content": imgURL}},
			{Tag: "meta", Attributes: map[string]string{"property": "og:url", "content": currentURL}},

			{Tag: "meta", Attributes: map[string]string{"name": "twitter:card", "content": "summary_large_image"}},
			{Tag: "meta", Attributes: map[string]string{"name": "twitter:site", "content": "@riverframework"}},

			{Tag: "link", TrustedAttributes: map[string]string{"rel": "icon", "href": Wave.GetPublicURL("favicon.svg")}},
		}, nil
	},
	GetRootTemplateData: func(r *http.Request) (map[string]any, error) {
		return map[string]any{
			"HTMLClass":                   theme.GetThemeData(r).HTMLClass,
			"SystemThemeScript":           theme.SystemThemeScript,
			"SystemThemeScriptSha256Hash": theme.SystemThemeScriptSha256Hash,
		}, nil
	},
}

//go:embed wave.config.json
var configBytes []byte

//go:embed all:__dist/static
var staticFS embed.FS

var Wave = wave.New(&wave.Config{
	ConfigBytes:            configBytes,
	StaticFS:               staticFS,
	StaticFSEmbedDirective: "all:__dist/static",
})

var Log = colorlog.New("app server")
