package river

import (
	"errors"
	"io/fs"
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

func NewLoadersRouter(options ...LoadersRouterOptions) *LoadersRouter {
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

func NewActionsRouter(options ...ActionsRouterOptions) *ActionsRouter {
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
					return validate.JSONBodyInto(r, iPtr)
				}
				return errors.New("unsupported method")
			},
		}),
		supportedMethods: supportedMethods,
	}
}

type RiverAppConfig struct {
	// Required -- the bytes of your wave.config.json file. You can use go:embed or
	// just read the file in yourself. Using go:embed is recommended for simpler
	// deployments and improved performance.
	WaveConfigJSON []byte

	// Required -- be sure to pass in a file system that has your <distDir>/static
	// directory as its ROOT. If you are using an embedded filesystem, you may need
	// to use fs.Sub to get the correct subdirectory. Using go:embed is recommended
	// for simpler deployments and improved performance.
	DistStaticFS fs.FS

	GetDefaultHeadEls    GetDefaultHeadElsFunc
	GetHeadElUniqueRules GetHeadElUniqueRulesFunc
	GetRootTemplateData  GetRootTemplateDataFunc
}

func NewRiverApp(o RiverAppConfig) *River {
	var rvr River

	rvr.Wave = wave.New(&wave.Config{
		WaveConfigJSON: o.WaveConfigJSON,
		DistStaticFS:   o.DistStaticFS,
	})

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

	return &rvr
}

type Loaders struct {
	river         *River
	loadersRouter *LoadersRouter
}
type Actions struct {
	river         *River
	actionsRouter *ActionsRouter
}

func (h *River) ServeStatic() func(http.Handler) http.Handler {
	return h.Wave.ServeStatic(true)
}

func (h *River) Loaders(loadersRouter *LoadersRouter) *Loaders {
	return &Loaders{
		river:         h,
		loadersRouter: loadersRouter,
	}
}
func (h *River) Actions(actionsRouter *ActionsRouter) *Actions {
	return &Actions{
		river:         h,
		actionsRouter: actionsRouter,
	}
}

func (h *Loaders) HandlerMountPattern() string {
	return "/*"
}
func (h *Loaders) Handler() http.Handler {
	return h.river.GetLoadersHandler(h.loadersRouter.NestedRouter)
}

func (h *Actions) HandlerMountPattern() string {
	return h.actionsRouter.MountRoot("*")
}
func (h *Actions) Handler() http.Handler {
	return h.river.GetActionsHandler(h.actionsRouter.Router)
}
func (h *Actions) SupportedMethods() map[string]bool {
	return h.actionsRouter.supportedMethods
}

type BuildRiverOptions struct {
	LoadersRouter *LoadersRouter
	ActionsRouter *ActionsRouter
	AdHocTypes    []*AdHocType
	ExtraTSCode   string
}

func (h *River) BuildRiver(o BuildRiverOptions) {
	h.Wave.BuildWaveWithHook(func(isDev bool) error {
		return h.buildInner(&buildInnerOptions{
			isDev:        isDev,
			buildOptions: &o,
		})
	})
}

type Route[I any, O any] = mux.Route[I, O]
type TaskHandler[I any, O any] = mux.TaskHandler[I, O]
