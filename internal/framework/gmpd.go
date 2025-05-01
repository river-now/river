package framework

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/river-now/river/kit/htmlutil"
	"github.com/river-now/river/kit/matcher"
	"github.com/river-now/river/kit/mux"
	"github.com/river-now/river/kit/response"
	"github.com/river-now/river/kit/tasks"
	"github.com/river-now/river/kit/typed"
	"golang.org/x/sync/errgroup"
)

/////////////////////////////////////////////////////////////////////
/////// MISC
/////////////////////////////////////////////////////////////////////

type SplatValues []string

var gmpdCache = typed.SyncMap[string, *cachedItemSubset]{}

type cachedItemSubset struct {
	ImportURLs      []string
	ExportKeys      []string
	ErrorExportKeys []string
	Deps            []string
}

/////////////////////////////////////////////////////////////////////
/////// CORE TYPES
/////////////////////////////////////////////////////////////////////

type ui_data_core struct {
	OutermostError    string `json:"outermostError,omitempty"`
	OutermostErrorIdx *int   `json:"outermostErrorIdx,omitempty"`
	ErrorExportKey    string `json:"errorExportKey,omitempty"`

	MatchedPatterns []string `json:"matchedPatterns,omitempty"`
	LoadersData     []any    `json:"loadersData,omitempty"`
	ImportURLs      []string `json:"importURLs,omitempty"`
	ExportKeys      []string `json:"exportKeys,omitempty"`
	HasRootData     bool     `json:"hasRootData,omitempty"`

	Params      mux.Params  `json:"params,omitempty"`
	SplatValues SplatValues `json:"splatValues,omitempty"`

	Deps []string `json:"deps,omitempty"`
}

type ui_data_stage_2 struct {
	Title string              `json:"title,omitempty"`
	Meta  []*htmlutil.Element `json:"metaHeadEls,omitempty"`
	Rest  []*htmlutil.Element `json:"restHeadEls,omitempty"`

	CSSBundles []string `json:"cssBundles,omitempty"`
	ViteDevURL string   `json:"viteDevURL,omitempty"`
}

type ui_data_all struct {
	notFound         bool
	didRedirect      bool
	didErr           bool
	ui_data_core     *ui_data_core
	stage_1_head_els []*htmlutil.Element
	state_2_final    *ui_data_stage_2
}

type final_ui_data struct {
	*ui_data_core
	*ui_data_stage_2
}

func (h *River) get_ui_data_stage_1(
	w http.ResponseWriter, r *http.Request, nestedRouter *mux.NestedRouter, tasksCtx *tasks.TasksCtx,
) *ui_data_all {
	realPath := matcher.StripTrailingSlash(r.URL.Path)
	if realPath == "" {
		realPath = "/"
	}

	_match_results, found := mux.FindNestedMatches(nestedRouter, r)
	if !found {
		return &ui_data_all{notFound: true}
	}

	_matches := _match_results.Matches

	matchedPatterns := make([]string, len(_matches))
	for i, match := range _matches {
		matchedPatterns[i] = match.OriginalPattern()
	}

	var sb strings.Builder
	var growSize int
	for _, match := range _matches {
		growSize += len(match.NormalizedPattern())
	}
	sb.Grow(growSize)
	for _, match := range _matches {
		sb.WriteString(match.NormalizedPattern())
	}
	cacheKey := sb.String()

	var _cachedItemSubset *cachedItemSubset
	var isCached bool

	if _cachedItemSubset, isCached = gmpdCache.Load(cacheKey); !isCached {
		_cachedItemSubset = &cachedItemSubset{}
		for _, path := range _matches {
			foundPath := h._paths[path.OriginalPattern()]
			if foundPath == nil {
				_cachedItemSubset.ImportURLs = append(_cachedItemSubset.ImportURLs, "")
				_cachedItemSubset.ExportKeys = append(_cachedItemSubset.ExportKeys, "")
				_cachedItemSubset.ErrorExportKeys = append(_cachedItemSubset.ErrorExportKeys, "")
				continue
			}
			pathToUse := foundPath.OutPath
			if h._isDev {
				pathToUse = foundPath.SrcPath
			}
			_cachedItemSubset.ImportURLs = append(_cachedItemSubset.ImportURLs, "/"+pathToUse)
			_cachedItemSubset.ExportKeys = append(_cachedItemSubset.ExportKeys, foundPath.ExportKey)
			_cachedItemSubset.ErrorExportKeys = append(_cachedItemSubset.ErrorExportKeys, foundPath.ErrorExportKey)
		}
		_cachedItemSubset.Deps = h.getDeps(_matches)
		_cachedItemSubset, _ = gmpdCache.LoadOrStore(cacheKey, _cachedItemSubset)
	}

	_tasks_results := mux.RunNestedTasks(nestedRouter, tasksCtx, r, _match_results)

	var hasRootData bool
	if len(_match_results.Matches) > 0 &&
		_match_results.Matches[0].NormalizedPattern() == "" &&
		_tasks_results.GetHasTaskHandler(0) {
		hasRootData = true
	}

	_merged_response_proxy := response.MergeProxyResponses(_tasks_results.ResponseProxies...)
	if _merged_response_proxy != nil {
		_merged_response_proxy.ApplyToResponseWriter(w, r)

		if _merged_response_proxy.IsError() {
			return &ui_data_all{didErr: true}
		}

		if _merged_response_proxy.IsRedirect() {
			return &ui_data_all{didRedirect: true}
		}
	}

	var numberOfLoaders int
	if _match_results != nil {
		numberOfLoaders = len(_match_results.Matches)
	}

	loadersData := make([]any, numberOfLoaders)
	loadersErrs := make([]error, numberOfLoaders)

	if numberOfLoaders > 0 {
		for i, result := range _tasks_results.Slice {
			if result != nil {
				loadersData[i] = result.Data()
				loadersErrs[i] = result.Err()
			}
		}
	}

	var outermostErrorIdx *int
	for i, err := range loadersErrs {
		if err != nil {
			outermostErrorIdx = &i
			break
		}
	}

	loadersHeadEls := make([][]*htmlutil.Element, numberOfLoaders)
	for _, _response_proxy := range _tasks_results.ResponseProxies {
		if _response_proxy != nil {
			loadersHeadEls = append(loadersHeadEls, _response_proxy.GetHeadElements())
		}
	}

	if outermostErrorIdx != nil {
		derefOuterMostErrorIdx := *outermostErrorIdx

		headElsDoubleSlice := loadersHeadEls[:derefOuterMostErrorIdx]
		headEls := make([]*htmlutil.Element, 0, len(headElsDoubleSlice))
		for _, slice := range headElsDoubleSlice {
			headEls = append(headEls, slice...)
		}

		cutIdx := derefOuterMostErrorIdx + 1

		ui_data := &ui_data_all{
			ui_data_core: &ui_data_core{
				OutermostError:    loadersErrs[derefOuterMostErrorIdx].Error(),
				OutermostErrorIdx: outermostErrorIdx,
				ErrorExportKey:    _cachedItemSubset.ErrorExportKeys[derefOuterMostErrorIdx],

				MatchedPatterns: matchedPatterns[:cutIdx],
				LoadersData:     loadersData[:cutIdx],
				ImportURLs:      _cachedItemSubset.ImportURLs[:cutIdx],
				ExportKeys:      _cachedItemSubset.ExportKeys[:cutIdx],
				HasRootData:     hasRootData,

				Params:      _match_results.Params,
				SplatValues: _match_results.SplatValues,

				Deps: _cachedItemSubset.Deps,
			},

			stage_1_head_els: headEls,
		}

		return ui_data
	}

	headEls := make([]*htmlutil.Element, 0, len(loadersHeadEls))
	for _, slice := range loadersHeadEls {
		headEls = append(headEls, slice...)
	}

	ui_data := &ui_data_all{
		ui_data_core: &ui_data_core{
			OutermostError:    "",
			OutermostErrorIdx: nil,

			MatchedPatterns: matchedPatterns,
			LoadersData:     loadersData,
			ImportURLs:      _cachedItemSubset.ImportURLs,
			ExportKeys:      _cachedItemSubset.ExportKeys,
			HasRootData:     hasRootData,

			Params:      _match_results.Params,
			SplatValues: _match_results.SplatValues,

			Deps: _cachedItemSubset.Deps,
		},

		stage_1_head_els: headEls,
	}

	return ui_data
}

func (h *River) getUIRouteData(w http.ResponseWriter, r *http.Request,
	nestedRouter *mux.NestedRouter,
) *ui_data_all {
	res := response.New(w)

	tasksCtx := nestedRouter.TasksRegistry().MustGetCtxFromRequest(r)

	eg := errgroup.Group{}

	var defaultHeadEls []*htmlutil.Element
	var err error

	eg.Go(func() error {
		if h.GetDefaultHeadEls != nil {
			defaultHeadEls, err = h.GetDefaultHeadEls(r)
			if err != nil {
				return fmt.Errorf("GetDefaultHeadEls error: %w", err)
			}
		} else {
			defaultHeadEls = []*htmlutil.Element{}
		}
		return nil
	})

	uiRoutesData := h.get_ui_data_stage_1(w, r, nestedRouter, tasksCtx)

	if uiRoutesData.notFound || uiRoutesData.didRedirect || uiRoutesData.didErr {
		return uiRoutesData
	}

	err = eg.Wait()
	if err != nil {
		Log.Error(err.Error())
		res.InternalServerError()
		return &ui_data_all{didErr: true}
	}

	var hb []*htmlutil.Element
	hb = make([]*htmlutil.Element, 0, len(uiRoutesData.stage_1_head_els)+len(defaultHeadEls))
	hb = append(hb, defaultHeadEls...)
	hb = append(hb, uiRoutesData.stage_1_head_els...)

	headEls := headElsInstance.ToSortedHeadEls(hb)

	ui_data := &ui_data_all{
		ui_data_core: uiRoutesData.ui_data_core,

		state_2_final: &ui_data_stage_2{
			Title: headEls.Title,
			Meta:  headEls.Meta,
			Rest:  headEls.Rest,

			CSSBundles: h.getCSSBundles(uiRoutesData.ui_data_core.Deps),
			ViteDevURL: h.getViteDevURL(),
		},
	}

	return ui_data
}
