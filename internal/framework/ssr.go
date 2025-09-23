package river

import (
	"fmt"
	"html/template"
	"path"
	"strings"

	"github.com/river-now/river/kit/envutil"
	"github.com/river-now/river/kit/htmlutil"
)

type SSRInnerHTMLInput struct {
	RiverSymbolStr string

	IsDev            bool
	ViteDevURL       string
	BuildID          string
	PublicPathPrefix string
	DeploymentID     string
	RouteManifestURL string

	*ui_data_core

	CSSBundles []string
}

// Sadly, must include the script tags so html/template parses this correctly.
// They are stripped off later in order to get the correct sha256 hash.
// Then they are added back via htmlutil.RenderElement.
const ssrInnerHTMLTmplStr = `<script>
globalThis[Symbol.for("{{.RiverSymbolStr}}")] = {};
const x = globalThis[Symbol.for("{{.RiverSymbolStr}}")];
x.patternToWaitFnMap = {};
x.clientLoadersData = [];
x.isDev = {{.IsDev}};
x.viteDevURL = {{.ViteDevURL}};
x.buildID = {{.BuildID}};
x.publicPathPrefix = "{{.PublicPathPrefix}}";
x.outermostError = {{.OutermostError}};
x.outermostErrorIdx = {{.OutermostErrorIdx}};
x.errorExportKey = {{.ErrorExportKey}};
x.matchedPatterns = {{.MatchedPatterns}};
x.loadersData = {{.LoadersData}};
x.importURLs = {{.ImportURLs}};
x.exportKeys = {{.ExportKeys}};
x.hasRootData = {{.HasRootData}};
x.params = {{.Params}};
x.splatValues = {{.SplatValues}};
x.deps = {{.Deps}};
x.cssBundles = {{.CSSBundles}};
x.deploymentID = {{.DeploymentID}};
x.routeManifestURL = {{.RouteManifestURL}};
</script>`

var ssrInnerTmpl = template.Must(template.New("ssr").Parse(ssrInnerHTMLTmplStr))

type GetSSRInnerHTMLOutput struct {
	Script     *template.HTML
	Sha256Hash string
}

func (h *River) getSSRInnerHTML(routeData *final_ui_data) (*GetSSRInnerHTMLOutput, error) {
	var htmlBuilder strings.Builder

	dto := SSRInnerHTMLInput{
		RiverSymbolStr: RiverSymbolStr,

		IsDev:            h._isDev,
		ViteDevURL:       routeData.ViteDevURL,
		BuildID:          h._buildID,
		PublicPathPrefix: h.Wave.GetPublicPathPrefix(),
		RouteManifestURL: path.Join(
			h.Wave.GetPublicPathPrefix(),
			h._routeManifestFile,
		),

		ui_data_core: routeData.ui_data_core,

		CSSBundles: routeData.CSSBundles,
	}

	if envutil.GetBool("VERCEL_SKEW_PROTECTION_ENABLED", false) {
		dto.DeploymentID = envutil.GetStr("VERCEL_DEPLOYMENT_ID", "")
	}

	if err := ssrInnerTmpl.Execute(&htmlBuilder, dto); err != nil {
		wrapped := fmt.Errorf("could not execute SSR inner HTML template: %w", err)
		Log.Error(wrapped.Error())
		return nil, wrapped
	}

	innerHTML := htmlBuilder.String()
	innerHTML = strings.TrimPrefix(innerHTML, "<script>")
	innerHTML = strings.TrimSuffix(innerHTML, "</script>")

	el := htmlutil.Element{
		Tag:                 "script",
		AttributesKnownSafe: map[string]string{"type": "module"},
		DangerousInnerHTML:  innerHTML,
	}

	sha256Hash, err := htmlutil.AddSha256HashInline(&el)
	if err != nil {
		wrapped := fmt.Errorf("could not handle CSP for SSR inner HTML: %w", err)
		Log.Error(wrapped.Error())
		return nil, wrapped
	}

	renderedEl, err := htmlutil.RenderElement(&el)
	if err != nil {
		wrapped := fmt.Errorf("could not render SSR inner HTML: %w", err)
		Log.Error(wrapped.Error())
		return nil, wrapped
	}

	return &GetSSRInnerHTMLOutput{Script: &renderedEl, Sha256Hash: sha256Hash}, nil
}
