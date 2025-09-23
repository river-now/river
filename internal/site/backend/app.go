package backend

import (
	"embed"
	"net/http"
	"path"

	"github.com/river-now/river"
	"github.com/river-now/river/kit/colorlog"
	"github.com/river-now/river/kit/fsutil"
	"github.com/river-now/river/kit/theme"
	"github.com/river-now/river/wave"
)

//go:embed all:dist/static wave.config.json
var embedFS embed.FS

var Log = colorlog.New("app server")

const (
	Domain          = "river.now"
	SiteTitle       = "River Framework"
	SiteDescription = "Vite-powered web framework bridging Go and TypeScript"
)

var App = river.NewRiverApp(river.RiverAppConfig{
	WaveConfigJSON: fsutil.MustReadFile(embedFS, "wave.config.json"),
	DistStaticFS:   fsutil.MustSub(embedFS, "dist", "static"),

	GetHeadElUniqueRules: func() *river.HeadEls {
		e := river.NewHeadEls(8)

		e.Meta(e.Property("og:title"))
		e.Meta(e.Property("og:description"))
		e.Meta(e.Property("og:type"))
		e.Meta(e.Property("og:image"))
		e.Meta(e.Property("og:url"))
		e.Meta(e.Name("twitter:card"))
		e.Meta(e.Name("twitter:site"))
		e.Link(e.Rel("icon"))

		return e
	},

	GetDefaultHeadEls: func(r *http.Request, app *river.River) (*river.HeadEls, error) {
		currentURL := "https://" + path.Join(Domain, r.URL.Path)

		ogImgURL := app.GetPublicURL("river-banner.webp")
		favURL := app.GetPublicURL("favicon.svg")

		if !wave.GetIsDev() {
			ogImgURL = "https://" + path.Join(Domain, ogImgURL)
		}

		e := river.NewHeadEls(12)

		e.Title(SiteTitle)
		e.Description(SiteDescription)

		e.Meta(e.Property("og:title"), e.Content(SiteTitle))
		e.Meta(e.Property("og:description"), e.Content(SiteDescription))
		e.Meta(e.Property("og:type"), e.Content("website"))
		e.Meta(e.Property("og:image"), e.Content(ogImgURL))
		e.Meta(e.Property("og:url"), e.Content(currentURL))

		e.Meta(e.Name("twitter:card"), e.Content("summary_large_image"))
		e.Meta(e.Name("twitter:site"), e.Content("@riverframework"))

		e.Link(e.Rel("icon"), e.Attr("href", favURL), e.Attr("type", "image/svg+xml"))

		for _, fontFile := range []string{
			"fonts/jetbrains_mono.woff2",
			"fonts/jetbrains_mono_italic.woff2",
		} {
			fontURL := app.GetPublicURL(fontFile)
			e.Link(
				e.Rel("preload"),
				e.Attr("as", "font"),
				e.Attr("type", "font/woff2"),
				e.Attr("crossorigin", "anonymous"),
				e.Attr("href", fontURL),
			)
		}

		return e, nil
	},

	GetRootTemplateData: func(r *http.Request) (map[string]any, error) {
		return map[string]any{
			"HTMLClass":                   theme.GetThemeData(r).HTMLClass,
			"SystemThemeScript":           theme.SystemThemeScript,
			"SystemThemeScriptSha256Hash": theme.SystemThemeScriptSha256Hash,
		}, nil
	},
})
