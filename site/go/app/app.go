package app

import (
	"embed"
	"net/http"

	"github.com/river-now/river"
	"github.com/river-now/river/kiruna"
	"github.com/river-now/river/kit/colorlog"
	"github.com/river-now/river/kit/theme"
)

const (
	Domain          = "river.now"
	Origin          = "https://" + Domain
	SiteTitle       = "River"
	SiteDescription = "River is a framework for writing modern, type-safe web applications with Go and TypeScript."
)

var River = &river.River{
	Kiruna: Kiruna,
	GetDefaultHeadBlocks: func(r *http.Request) ([]*river.HeadBlock, error) {
		imgURL := Kiruna.GetPublicURL("river-banner.webp")

		return []*river.HeadBlock{
			{Tag: "title", InnerHTML: SiteTitle},
			{Tag: "meta", Attributes: map[string]string{"name": "description", "content": SiteDescription}},

			{Tag: "meta", Attributes: map[string]string{"property": "og:url", "content": Origin + r.URL.Path}},
			{Tag: "meta", Attributes: map[string]string{"property": "og:type", "content": "website"}},
			{Tag: "meta", Attributes: map[string]string{"property": "og:title", "content": SiteTitle}},
			{Tag: "meta", Attributes: map[string]string{"property": "og:description", "content": SiteDescription}},
			{Tag: "meta", Attributes: map[string]string{"property": "og:image", "content": imgURL}},

			{Tag: "meta", Attributes: map[string]string{"name": "twitter:card", "content": "summary_large_image"}},
			{Tag: "meta", Attributes: map[string]string{"property": "twitter:domain", "content": Domain}},
			{Tag: "meta", Attributes: map[string]string{"property": "twitter:url", "content": Origin + r.URL.Path}},
			{Tag: "meta", Attributes: map[string]string{"name": "twitter:title", "content": SiteTitle}},
			{Tag: "meta", Attributes: map[string]string{"name": "twitter:description", "content": SiteDescription}},
			{Tag: "meta", Attributes: map[string]string{"name": "twitter:image", "content": imgURL}},

			{Tag: "link", TrustedAttributes: map[string]string{"rel": "icon", "href": Kiruna.GetPublicURL("favicon.svg")}},
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

//go:embed kiruna.config.json
var configBytes []byte

//go:embed all:kiruna_dist/static
var staticFS embed.FS

var Kiruna = kiruna.New(&kiruna.Config{
	ConfigBytes:            configBytes,
	StaticFS:               staticFS,
	StaticFSEmbedDirective: "all:kiruna_dist/static",
})

var Log = colorlog.New("app server")
