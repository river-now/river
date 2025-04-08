package framework

import (
	"fmt"
	"net/http"

	"github.com/sjc5/river/kit/htmlutil"
	"github.com/sjc5/river/kit/lru"
	"github.com/sjc5/river/kit/matcher"
	"github.com/sjc5/river/kit/mux"
	"github.com/sjc5/river/kit/response"
	"github.com/sjc5/river/kit/tasks"
)

type SplatValues []string

type ActivePathData struct {
	HeadBlocks  []*htmlutil.Element
	LoadersData []any
	// LoadersErrMsgs      []string
	LoadersErrs         []error
	ImportURLs          []string
	ExportKeys          []string
	OutermostErrorIndex int
	SplatValues         SplatValues
	Params              mux.Params
	Deps                []string
	HasRootData         bool
}

type gmpdItem struct {
	found          bool
	_match_results *matcher.FindNestedMatchesResults
	Params         mux.Params
	SplatValues    SplatValues
	ImportURLs     []string
	ExportKeys     []string
	Deps           []string
	routeType      RouteType
}

var gmpdCache = lru.NewCache[string, *gmpdItem](500_000)

type uiRoutesData struct {
	activePathData *ActivePathData
	didRedirect    bool
	didErr         bool
	found          bool
}

// Returns nil if no match is found
func (h *River) getUIRoutesData(
	w http.ResponseWriter, r *http.Request, nestedRouter *mux.NestedRouter, tasksCtx *tasks.TasksCtx,
) *uiRoutesData {

	realPath := matcher.StripTrailingSlash(r.URL.Path)
	if realPath == "" {
		realPath = "/"
	}

	var itemIsCached bool
	var item *gmpdItem

	if item, itemIsCached = gmpdCache.Get(realPath); !itemIsCached {
		item = new(gmpdItem)
		_match_results, found := mux.FindNestedMatches(nestedRouter, r)
		if !found {
			item.found = false
			gmpdCache.Set(realPath, item, true)
			return &uiRoutesData{}
		}
		item.found = true
		item._match_results = _match_results
		_matches := _match_results.Matches
		_matches_len := len(_matches)
		item.SplatValues = _match_results.SplatValues
		item.Params = _match_results.Params
		item.routeType = RouteTypes.Loader
		item.ImportURLs = make([]string, 0, _matches_len)
		item.ExportKeys = make([]string, 0, _matches_len)
		for _, path := range _matches {
			foundPath := h._paths[path.OriginalPattern()]
			if foundPath == nil {
				continue
			}
			pathToUse := foundPath.OutPath
			if h._isDev {
				pathToUse = foundPath.SrcPath
			}
			item.ImportURLs = append(item.ImportURLs, "/"+pathToUse)
			item.ExportKeys = append(item.ExportKeys, foundPath.ExportKey)
		}
		item.Deps = h.getDeps(_matches)
		gmpdCache.Set(realPath, item, false)
	}

	if !item.found {
		return &uiRoutesData{}
	}

	_tasks_results := mux.RunNestedTasks(nestedRouter, tasksCtx, r, item._match_results)

	var hasRootData bool
	if len(item._match_results.Matches) > 0 &&
		item._match_results.Matches[0].NormalizedPattern() == "" &&
		_tasks_results.GetHasTaskHandler(0) {
		hasRootData = true
	}

	_merged_response_proxy := response.MergeProxyResponses(_tasks_results.ResponseProxies...)
	if _merged_response_proxy != nil {
		_merged_response_proxy.ApplyToResponseWriter(w, r)

		if _merged_response_proxy.IsError() {
			return &uiRoutesData{didErr: true, found: true}
		}

		if _merged_response_proxy.IsRedirect() {
			return &uiRoutesData{didRedirect: true, found: true}
		}
	}

	var numberOfLoaders int
	if item._match_results != nil {
		numberOfLoaders = len(item._match_results.Matches)
	}

	loadersData := make([]any, numberOfLoaders)
	// loadersErrMsgs := make([]string, numberOfLoaders)
	loadersErrs := make([]error, numberOfLoaders)

	if numberOfLoaders > 0 {
		for i, result := range _tasks_results.Slice {
			if result != nil {
				loadersData[i] = result.Data()
				loadersErrs[i] = result.Err()
			}
		}
	}

	var thereAreErrors bool
	outermostErrorIndex := -1
	for i, err := range loadersErrs {
		if err != nil {
			Log.Error(fmt.Sprintf("ERROR: %s", err))
			thereAreErrors = true
			outermostErrorIndex = i
			break
		}
	}

	loadersHeadBlocks := make([][]*htmlutil.Element, numberOfLoaders)
	for _, _response_proxy := range _tasks_results.ResponseProxies {
		if _response_proxy != nil {
			loadersHeadBlocks = append(loadersHeadBlocks, _response_proxy.GetHeadElements())
		}
	}

	if thereAreErrors && item.routeType == RouteTypes.Loader {
		headBlocksDoubleSlice := loadersHeadBlocks[:outermostErrorIndex]
		headblocks := make([]*htmlutil.Element, 0, len(headBlocksDoubleSlice))
		for _, slice := range headBlocksDoubleSlice {
			headblocks = append(headblocks, slice...)
		}

		apd := &ActivePathData{
			LoadersData:         loadersData[:outermostErrorIndex],
			ImportURLs:          item.ImportURLs[:outermostErrorIndex+1],
			ExportKeys:          item.ExportKeys[:outermostErrorIndex+1],
			OutermostErrorIndex: outermostErrorIndex,
			SplatValues:         item.SplatValues,
			Params:              item.Params,
			// LoadersErrMsgs:      loadersErrs[:outermostErrorIndex+1],
			LoadersErrs: loadersErrs[:outermostErrorIndex+1],
			HeadBlocks:  headblocks,
			HasRootData: hasRootData,
		}

		return &uiRoutesData{activePathData: apd, found: true}
	}

	headblocks := make([]*htmlutil.Element, 0, len(loadersHeadBlocks))
	for _, slice := range loadersHeadBlocks {
		headblocks = append(headblocks, slice...)
	}

	apd := &ActivePathData{
		LoadersData:         loadersData,
		ImportURLs:          item.ImportURLs,
		ExportKeys:          item.ExportKeys,
		OutermostErrorIndex: outermostErrorIndex,
		SplatValues:         item.SplatValues,
		Params:              item.Params,
		Deps:                item.Deps,
		// LoadersErrMsgs:      loadersErrMsgs,
		LoadersErrs: loadersErrs,
		HeadBlocks:  headblocks,
		HasRootData: hasRootData,
	}

	return &uiRoutesData{activePathData: apd, found: true}
}
