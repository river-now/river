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
	Domain          = ""
	Origin          = "https://" + Domain
	SiteTitle       = "River"
	SiteDescription = "River is a framework for writing modern, type-safe web applications with Go and TypeScript."
)

var River = &river.River{
	Kiruna: Kiruna,
	GetDefaultHeadBlocks: func(r *http.Request) ([]*river.HeadBlock, error) {
		return []*river.HeadBlock{
			{Tag: "title", InnerHTML: SiteTitle},
			{Tag: "meta", Attributes: map[string]string{"name": "description", "content": SiteDescription}},
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
