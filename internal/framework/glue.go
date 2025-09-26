package river

import (
	"errors"
	"mime"
	"net/http"

	"github.com/river-now/river/kit/headels"
	"github.com/river-now/river/kit/mux"
	"github.com/river-now/river/kit/validate"
	"github.com/river-now/river/wave"
)

type LoadersRouter struct {
	*mux.NestedRouter
}
type ActionsRouter struct {
	*mux.Router
	supportedMethods map[string]bool
}
type LoaderReqData = mux.NestedReqData
type ActionReqData[I any] = mux.ReqData[I]

type LoadersRouterOptions struct {
	// Default: ':' (e.g., /user/:id)
	DynamicParamPrefix rune
	// Default: '*' (e.g., /files/*)
	SplatSegmentIdentifier rune
	// Default: "_index" (e.g., /blog/_index)
	IndexSegmentIdentifier string
}

type ActionsRouterOptions struct {
	// Default: ':' (e.g., /user/:id)
	DynamicParamPrefix rune
	// Default: '*' (e.g., /files/*)
	SplatSegmentIdentifier rune
	// Default: "/api/"
	MountRoot string
	// Default: []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	SupportedMethods []string
}

func newLoadersRouter(options ...LoadersRouterOptions) *LoadersRouter {
	var o LoadersRouterOptions
	if len(options) > 0 {
		o = options[0]
	}
	explicitIndexSegment := o.IndexSegmentIdentifier
	if explicitIndexSegment == "" {
		explicitIndexSegment = "_index"
	}

	return &LoadersRouter{
		NestedRouter: mux.NewNestedRouter(&mux.NestedOptions{
			DynamicParamPrefixRune: o.DynamicParamPrefix,
			SplatSegmentRune:       o.SplatSegmentIdentifier,
			ExplicitIndexSegment:   explicitIndexSegment,
		}),
	}
}

func newActionsRouter(options ...ActionsRouterOptions) *ActionsRouter {
	var o ActionsRouterOptions
	if len(options) > 0 {
		o = options[0]
	}

	mountRoot := o.MountRoot
	if mountRoot == "" {
		mountRoot = "/api/"
	}

	supportedMethods := make(map[string]bool, len(o.SupportedMethods))
	if len(o.SupportedMethods) == 0 {
		supportedMethods["GET"] = true
		supportedMethods["POST"] = true
		supportedMethods["PUT"] = true
		supportedMethods["DELETE"] = true
		supportedMethods["PATCH"] = true
	} else {
		for _, m := range o.SupportedMethods {
			supportedMethods[m] = true
		}
	}

	return &ActionsRouter{
		Router: mux.NewRouter(&mux.Options{
			DynamicParamPrefixRune: o.DynamicParamPrefix,
			SplatSegmentRune:       o.SplatSegmentIdentifier,
			MountRoot:              mountRoot,
			ParseInput: func(r *http.Request, iPtr any) error {
				if r.Method == http.MethodGet {
					return validate.URLSearchParamsInto(r, iPtr)
				}
				if supportedMethods[r.Method] {
					contentType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
					if contentType == "application/x-www-form-urlencoded" ||
						contentType == "multipart/form-data" {
						return nil
					}
					return validate.JSONBodyInto(r, iPtr)
				}
				return errors.New("unsupported method")
			},
		}),
		supportedMethods: supportedMethods,
	}
}

type FormData struct{}

func (m FormData) TSTypeRaw() string { return "FormData" }

type RiverAppConfig struct {
	Wave *wave.Wave

	GetDefaultHeadEls    GetDefaultHeadElsFunc
	GetHeadElUniqueRules GetHeadElUniqueRulesFunc
	GetRootTemplateData  GetRootTemplateDataFunc

	LoadersRouterOptions LoadersRouterOptions
	ActionsRouterOptions ActionsRouterOptions
}

func NewRiverApp(o RiverAppConfig) *River {
	var rvr River

	rvr.Wave = o.Wave
	if rvr.Wave == nil {
		panic("Wave instance is required")
	}

	rvr.getDefaultHeadEls = o.GetDefaultHeadEls
	if rvr.getDefaultHeadEls == nil {
		rvr.getDefaultHeadEls = func(r *http.Request, app *River) (*headels.HeadEls, error) {
			return headels.New(), nil
		}
	}

	rvr.getHeadElUniqueRules = o.GetHeadElUniqueRules
	if rvr.getHeadElUniqueRules == nil {
		rvr.getHeadElUniqueRules = func() *headels.HeadEls {
			return headels.New()
		}
	}

	rvr.getRootTemplateData = o.GetRootTemplateData
	if rvr.getRootTemplateData == nil {
		rvr.getRootTemplateData = func(r *http.Request) (map[string]any, error) {
			return map[string]any{}, nil
		}
	}

	rvr.loadersRouter = newLoadersRouter(o.LoadersRouterOptions)
	rvr.actionsRouter = newActionsRouter(o.ActionsRouterOptions)

	return &rvr
}

type Loaders struct{ river *River }
type Actions struct{ river *River }

func (h *River) ServeStatic() func(http.Handler) http.Handler {
	return h.Wave.ServeStatic(true)
}

func (h *River) Loaders() *Loaders { return &Loaders{river: h} }
func (h *River) Actions() *Actions { return &Actions{river: h} }

func (h *Loaders) HandlerMountPattern() string {
	return "/*"
}
func (h *Loaders) Handler() http.Handler {
	return h.river.GetLoadersHandler(h.river.LoadersRouter().NestedRouter)
}

func (h *Actions) HandlerMountPattern() string {
	return h.river.ActionsRouter().MountRoot("*")
}
func (h *Actions) Handler() http.Handler {
	return h.river.GetActionsHandler(h.river.ActionsRouter().Router)
}
func (h *Actions) SupportedMethods() map[string]bool {
	return h.river.ActionsRouter().supportedMethods
}

type BuildOptions struct {
	AdHocTypes  []*AdHocType
	ExtraTSCode string
}

func (h *River) Build(o ...BuildOptions) {
	var opts BuildOptions
	if len(o) > 0 {
		opts = o[0]
	}
	h.Wave.BuildWaveWithHook(func(isDev bool) error {
		return h.buildInner(&buildInnerOptions{
			isDev:        isDev,
			buildOptions: &opts,
		})
	})
}

type Route[I any, O any] = mux.Route[I, O]
type TaskHandler[I any, O any] = mux.TaskHandler[I, O]
