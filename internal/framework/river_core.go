package river

import (
	"html/template"
	"io/fs"
	"net/http"
	"sync"

	"github.com/river-now/river/kit/colorlog"
	"github.com/river-now/river/kit/headels"
	"github.com/river-now/river/kit/mux"
	"github.com/river-now/river/wave"
)

const (
	RiverSymbolStr = "__river_internal__"
)

var Log = colorlog.New("river")

type RouteType = string

var RouteTypes = struct {
	Loader   RouteType
	Query    RouteType
	Mutation RouteType
	NotFound RouteType
}{
	Loader:   "loader",
	Query:    "query",
	Mutation: "mutation",
	NotFound: "not-found",
}

type Path struct {
	NestedRoute mux.AnyNestedRoute `json:"-"`

	// both stages one and two
	OriginalPattern string `json:"originalPattern"`
	SrcPath         string `json:"srcPath"`
	ExportKey       string `json:"exportKey"`
	ErrorExportKey  string `json:"errorExportKey,omitempty"`

	// stage two only
	OutPath string   `json:"outPath,omitempty"`
	Deps    []string `json:"deps,omitempty"`
}

type UIVariant string

var UIVariants = struct {
	React  UIVariant
	Preact UIVariant
	Solid  UIVariant
}{
	React:  "react",
	Preact: "preact",
	Solid:  "solid",
}

type (
	GetDefaultHeadElsFunc    func(r *http.Request, app *River) (*headels.HeadEls, error)
	GetHeadElUniqueRulesFunc func() *headels.HeadEls
	GetRootTemplateDataFunc  func(r *http.Request) (map[string]any, error)
)

type River struct {
	*wave.Wave

	getDefaultHeadEls    GetDefaultHeadElsFunc
	getHeadElUniqueRules GetHeadElUniqueRulesFunc
	getRootTemplateData  GetRootTemplateDataFunc

	mu                 sync.RWMutex
	_isDev             bool
	_paths             map[string]*Path
	_clientEntrySrc    string
	_clientEntryOut    string
	_clientEntryDeps   []string
	_buildID           string
	_depToCSSBundleMap map[string]string
	_rootTemplate      *template.Template
	_privateFS         fs.FS
	_routeManifestFile string
	_serverAddr        string
}

func (h *River) ServerAddr() string {
	return h._serverAddr
}
