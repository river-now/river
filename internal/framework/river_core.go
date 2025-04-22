package framework

import (
	"html/template"
	"io/fs"
	"net/http"
	"sync"

	"github.com/river-now/river/kit/colorlog"
	"github.com/river-now/river/kit/headels"
	"github.com/river-now/river/kit/htmlutil"
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
	Pattern        string `json:"pattern"`
	SrcPath        string `json:"srcPath"`
	ExportKey      string `json:"exportKey"`
	ErrorExportKey string `json:"errorExportKey,omitempty"`

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

type River struct {
	Wave                 *wave.Wave
	GetDefaultHeadEls    func(r *http.Request) ([]*htmlutil.Element, error)
	GetHeadElUniqueRules func() *headels.HeadEls
	GetRootTemplateData  func(r *http.Request) (map[string]any, error)

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
}
