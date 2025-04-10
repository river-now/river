package app

import (
	"embed"
	"net/http"

	"github.com/sjc5/river"
	"github.com/sjc5/river/kiruna"
	"github.com/sjc5/river/kit/colorlog"
	"github.com/sjc5/river/kit/theme"
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

//go:embed all:kiruna_dist/static
var embedFS embed.FS

var Kiruna = kiruna.New(&kiruna.Config{
	ConfigFile:     "./kiruna.config.json",
	EmbedDirective: "all:kiruna_dist/static",
	DistFS:         embedFS,
})

var Log = colorlog.New("app server")
