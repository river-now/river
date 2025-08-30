package framework

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	"github.com/river-now/river/kit/headels"
	"github.com/river-now/river/kit/mux"
	"github.com/river-now/river/kit/response"
	"github.com/river-now/river/kit/viteutil"
	"golang.org/x/sync/errgroup"
)

const RiverBuildIDHeaderKey = "X-River-Build-Id"

var headElsInstance = headels.NewInstance("river")

// Deprecated: use GetLoadersHandler instead.
func (h *River) GetUIHandler(nestedRouter *mux.NestedRouter) mux.TasksCtxRequirerFunc {
	return h.GetLoadersHandler(nestedRouter)
}

func (h *River) GetLoadersHandler(nestedRouter *mux.NestedRouter) mux.TasksCtxRequirerFunc {
	h.validateAndDecorateNestedRouter(nestedRouter)

	handler := mux.TasksCtxRequirerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := response.New(w)
		res.SetHeader(RiverBuildIDHeaderKey, h._buildID)

		isJSON := IsJSONRequest(r)
		if isJSON && !h.IsCurrentBuildJSONRequest(r) {
			newURL, err := url.Parse(r.URL.Path)
			if err != nil {
				Log.Error(fmt.Sprintf("Error parsing URL: %v\n", err))
				res.InternalServerError()
				return
			}
			q := newURL.Query()
			q.Del("river_json")
			newURL.RawQuery = q.Encode()
			res.SetHeader("X-River-Reload", newURL.String())
			res.OK()
			return
		}

		uiRouteData := h.getUIRouteData(w, r, nestedRouter)

		if uiRouteData.notFound {
			res.NotFound()
			return
		}

		if uiRouteData.didErr || uiRouteData.didRedirect {
			return
		}

		routeData := &final_ui_data{
			ui_data_core: uiRouteData.ui_data_core,
			Title:        uiRouteData.state_2_final.SortedAndPreEscapedHeadEls.Title,
			Meta:         uiRouteData.state_2_final.SortedAndPreEscapedHeadEls.Meta,
			Rest:         uiRouteData.state_2_final.SortedAndPreEscapedHeadEls.Rest,
			CSSBundles:   uiRouteData.state_2_final.CSSBundles,
			ViteDevURL:   uiRouteData.state_2_final.ViteDevURL,
		}

		currentCacheControlHeader := w.Header().Get("Cache-Control")

		if currentCacheControlHeader == "" {
			// Set a conservative default cache control header
			res.SetHeader("Cache-Control", "private, max-age=0, must-revalidate, no-cache")
		}

		if isJSON {
			jsonBytes, err := json.Marshal(routeData)
			if err != nil {
				Log.Error(fmt.Sprintf("Error marshalling JSON: %v\n", err))
				res.InternalServerError()
				return
			}

			res.JSONBytes(jsonBytes)
			return
		}

		var eg errgroup.Group
		var ssrScript *template.HTML
		var ssrScriptSha256Hash string
		var headElements template.HTML

		eg.Go(func() error {
			he, err := headElsInstance.Render(uiRouteData.state_2_final.SortedAndPreEscapedHeadEls)
			if err != nil {
				return fmt.Errorf("error getting head elements: %w", err)
			}
			headElements = he
			headElements += "\n" + h.Wave.GetCriticalCSSStyleElement()
			headElements += "\n" + h.Wave.GetStyleSheetLinkElement()

			return nil
		})

		eg.Go(func() error {
			sih, err := h.GetSSRInnerHTML(routeData)
			if err != nil {
				return fmt.Errorf("error getting SSR inner HTML: %w", err)
			}
			ssrScript = sih.Script
			ssrScriptSha256Hash = sih.Sha256Hash
			return nil
		})

		if err := eg.Wait(); err != nil {
			Log.Error(fmt.Sprintf("Error getting route data: %v\n", err))
			res.InternalServerError()
			return
		}

		var rootTemplateData map[string]any
		var err error
		if h.GetRootTemplateData != nil {
			rootTemplateData, err = h.GetRootTemplateData(r)
		} else {
			rootTemplateData = make(map[string]any)
		}
		if err != nil {
			Log.Error(fmt.Sprintf("Error getting root template data: %v\n", err))
			res.InternalServerError()
			return
		}

		rootTemplateData["RiverHeadEls"] = headElements
		rootTemplateData["RiverSSRScript"] = ssrScript
		rootTemplateData["RiverSSRScriptSha256Hash"] = ssrScriptSha256Hash
		rootTemplateData["RiverRootID"] = "river-root"

		if !h._isDev {
			rootTemplateData["RiverBodyScripts"] = template.HTML(
				fmt.Sprintf(
					`<script type="module" src="%s%s"></script>`,
					h.Wave.GetPublicPathPrefix(), h._clientEntryOut,
				),
			)
		} else {
			opts := viteutil.ToDevScriptsOptions{ClientEntry: h._clientEntrySrc}
			if UIVariant(h.Wave.GetRiverUIVariant()) == UIVariants.React {
				opts.Variant = viteutil.Variants.React
			} else {
				opts.Variant = viteutil.Variants.Other
			}

			devScripts, err := viteutil.ToDevScripts(opts)
			if err != nil {
				Log.Error(fmt.Sprintf("Error getting dev scripts: %v\n", err))
				res.InternalServerError()
				return
			}

			rootTemplateData["RiverBodyScripts"] = devScripts + "\n" + h.Wave.GetRefreshScript()
		}

		var buf bytes.Buffer

		err = h._rootTemplate.Execute(&buf, rootTemplateData)
		if err != nil {
			Log.Error(fmt.Sprintf("Error executing template: %v\n", err))
			res.InternalServerError()
		}

		res.HTMLBytes(buf.Bytes())
	})

	return handler
}

// If true, is JSON, but may or may not be from an up-to-date client.
func IsJSONRequest(r *http.Request) bool {
	return r.URL.Query().Get("river_json") != ""
}

// If true, is both (1) JSON and (2) guaranteed to be from a client
// that has knowledge of the latest build ID.
func (h *River) IsCurrentBuildJSONRequest(r *http.Request) bool {
	return r.URL.Query().Get("river_json") == h._buildID
}

func (h *River) GetCurrentBuildID() string {
	return h._buildID
}

func (h *River) GetActionsHandler(router *mux.Router) mux.TasksCtxRequirerFunc {
	return mux.TasksCtxRequirerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := response.New(w)
		res.SetHeader(RiverBuildIDHeaderKey, h._buildID)
		router.ServeHTTP(w, r)
	})
}
