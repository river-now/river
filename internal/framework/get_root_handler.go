package framework

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	"github.com/river-now/river/kit/headblocks"
	"github.com/river-now/river/kit/mux"
	"github.com/river-now/river/kit/response"
	"github.com/river-now/river/kit/viteutil"
	"golang.org/x/sync/errgroup"
)

const buildIDHeader = "X-River-Build-Id"

var headblocksInstance = headblocks.New("river")

func (h *River) GetUIHandler(nestedRouter *mux.NestedRouter) http.Handler {
	h.validateAndDecorateNestedRouter(nestedRouter)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := response.New(w)
		res.SetHeader(buildIDHeader, h._buildID)

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

		uiRouteData, err := h.getUIRouteData(w, r, nestedRouter)

		if err != nil && isErrNotFound(err) {
			Log.Error("Not found", "path", r.URL.Path)
			res.NotFound()
			return
		}

		if uiRouteData.didRedirect {
			return
		}

		if err != nil {
			Log.Error(fmt.Sprintf("Error getting route data: %v\n", err))
			res.InternalServerError()
			return
		}

		routeData := uiRouteData.uiRouteOutput

		currentCacheControlHeader := w.Header().Get("Cache-Control")

		if currentCacheControlHeader == "" {
			if isJSON {
				res.SetHeader("Cache-Control", "private, max-age=0, must-revalidate, no-cache")
			} else {
				res.SetHeader("Cache-Control", "private, max-age=0, must-revalidate, no-cache, no-store")
			}
		}

		if isJSON {
			jsonBytes, err := json.Marshal(routeData)
			if err != nil {
				Log.Error(fmt.Sprintf("Error marshalling JSON: %v\n", err))
				res.InternalServerError()
				return
			}

			if h.Kiruna.GetRiverAutoETags() {
				hashInput := []byte(r.Header.Get("Cookie"))
				if len(hashInput) > 4096 {
					Log.Error("Cookie too large")
					res.InternalServerError()
					return
				}
				hashInput = append(hashInput, jsonBytes...)
				etag := response.ToQuotedSha256Etag(hashInput)
				res.SetETag(etag)
				if response.ShouldReturn304Conservative(r, etag) {
					res.NotModified()
					return
				}
			}

			res.JSONBytes(jsonBytes)
			return
		}

		var eg errgroup.Group
		var ssrScript *template.HTML
		var ssrScriptSha256Hash string
		var headElements template.HTML

		eg.Go(func() error {
			he, err := headblocksInstance.Render(&headblocks.HeadBlocks{
				Title: routeData.Title,
				Meta:  routeData.Meta,
				Rest:  routeData.Rest,
			})
			if err != nil {
				return fmt.Errorf("error getting head elements: %w", err)
			}
			headElements = he
			headElements += "\n" + h.Kiruna.GetCriticalCSSStyleElement()
			headElements += "\n" + h.Kiruna.GetStyleSheetLinkElement()

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

		rootTemplateData["RiverHeadBlocks"] = headElements
		rootTemplateData["RiverSSRScript"] = ssrScript
		rootTemplateData["RiverSSRScriptSha256Hash"] = ssrScriptSha256Hash
		rootTemplateData["RiverRootID"] = "river-root"

		if !h._isDev {
			rootTemplateData["RiverBodyScripts"] = template.HTML(
				fmt.Sprintf(`<script type="module" src="%s%s"></script>`, h.Kiruna.GetPublicPathPrefix(), h._clientEntryOut),
			)
		} else {
			opts := viteutil.ToDevScriptsOptions{ClientEntry: h._clientEntrySrc}
			if UIVariant(h.Kiruna.GetRiverUIVariant()) == UIVariants.React {
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

			rootTemplateData["RiverBodyScripts"] = devScripts + "\n" + h.Kiruna.GetRefreshScript()
		}

		var buf bytes.Buffer

		err = h._rootTemplate.Execute(&buf, rootTemplateData)
		if err != nil {
			Log.Error(fmt.Sprintf("Error executing template: %v\n", err))
			res.InternalServerError()
		}

		res.HTMLBytes(buf.Bytes())
	})
}

// If true, may or may not be from an up-to-date client.
func IsJSONRequest(r *http.Request) bool {
	return r.URL.Query().Get("river_json") != ""
}

// If true, guaranteed to be from a client with knowledge of the latest build ID.
func (h *River) IsCurrentBuildJSONRequest(r *http.Request) bool {
	return r.URL.Query().Get("river_json") == h._buildID
}

func (h *River) GetActionsHandler(router *mux.Router) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := response.New(w)
		res.SetHeader(buildIDHeader, h._buildID)
		router.ServeHTTP(w, r)
	})
}
