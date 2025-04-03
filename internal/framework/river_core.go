package framework

import (
	"html/template"
	"io/fs"
	"net/http"
	"sync"

	"github.com/sjc5/river/kiruna"
	"github.com/sjc5/river/kit/colorlog"
	"github.com/sjc5/river/kit/genericsutil"
	"github.com/sjc5/river/kit/htmlutil"
	"github.com/sjc5/river/kit/mux"
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
	Pattern   string `json:"pattern"`
	SrcPath   string `json:"srcPath"`
	ExportKey string `json:"exportKey"`

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

type River[C any] struct {
	Kiruna               *kiruna.Kiruna
	GetDefaultHeadBlocks func(r *http.Request) ([]*htmlutil.Element, error)
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

type RiverAny interface{ _get_core_data_zero() any }

func (h *River[C]) _get_core_data_zero() any {
	return genericsutil.Zero[C]()
}
