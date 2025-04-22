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
)

type SplatValues []string

type ActivePathData struct {
	HeadEls     []*htmlutil.Element
	LoadersData []any
	// LoadersErrMsgs      []string
	LoadersErrs         []error
	ImportURLs          []string
	ExportKeys          []string
	OutermostErrorIndex int
	MatchedPatterns     []string
	SplatValues         SplatValues
	Params              mux.Params
	Deps                []string
	HasRootData         bool
}

type cachedItemSubset struct {
	ImportURLs []string
	ExportKeys []string
	Deps       []string
}

var gmpdCache = typed.SyncMap[string, *cachedItemSubset]{}

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

	_match_results, found := mux.FindNestedMatches(nestedRouter, r)
	if !found {
		return &uiRoutesData{}
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
				continue
			}
			pathToUse := foundPath.OutPath
			if h._isDev {
				pathToUse = foundPath.SrcPath
			}
			_cachedItemSubset.ImportURLs = append(_cachedItemSubset.ImportURLs, "/"+pathToUse)
			_cachedItemSubset.ExportKeys = append(_cachedItemSubset.ExportKeys, foundPath.ExportKey)
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
			return &uiRoutesData{didErr: true, found: true}
		}

		if _merged_response_proxy.IsRedirect() {
			return &uiRoutesData{didRedirect: true, found: true}
		}
	}

	var numberOfLoaders int
	if _match_results != nil {
		numberOfLoaders = len(_match_results.Matches)
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

	loadersHeadEls := make([][]*htmlutil.Element, numberOfLoaders)
	for _, _response_proxy := range _tasks_results.ResponseProxies {
		if _response_proxy != nil {
			loadersHeadEls = append(loadersHeadEls, _response_proxy.GetHeadElements())
		}
	}

	if thereAreErrors {
		headElsDoubleSlice := loadersHeadEls[:outermostErrorIndex]
		headEls := make([]*htmlutil.Element, 0, len(headElsDoubleSlice))
		for _, slice := range headElsDoubleSlice {
			headEls = append(headEls, slice...)
		}

		apd := &ActivePathData{
			LoadersData: loadersData[:outermostErrorIndex],
			// LoadersErrMsgs:      loadersErrs[:outermostErrorIndex+1],
			ImportURLs:          _cachedItemSubset.ImportURLs[:outermostErrorIndex+1],
			ExportKeys:          _cachedItemSubset.ExportKeys[:outermostErrorIndex+1],
			OutermostErrorIndex: outermostErrorIndex,
			MatchedPatterns:     matchedPatterns[:outermostErrorIndex+1],
			SplatValues:         _match_results.SplatValues,
			Params:              _match_results.Params,
			Deps:                _cachedItemSubset.Deps,
			LoadersErrs:         loadersErrs[:outermostErrorIndex+1],
			HeadEls:             headEls,
			HasRootData:         hasRootData,
		}

		return &uiRoutesData{activePathData: apd, found: true}
	}

	headEls := make([]*htmlutil.Element, 0, len(loadersHeadEls))
	for _, slice := range loadersHeadEls {
		headEls = append(headEls, slice...)
	}

	apd := &ActivePathData{
		LoadersData: loadersData,
		// LoadersErrMsgs:      loadersErrMsgs,
		ImportURLs:          _cachedItemSubset.ImportURLs,
		ExportKeys:          _cachedItemSubset.ExportKeys,
		OutermostErrorIndex: outermostErrorIndex,
		MatchedPatterns:     matchedPatterns,
		SplatValues:         _match_results.SplatValues,
		Params:              _match_results.Params,
		Deps:                _cachedItemSubset.Deps,
		LoadersErrs:         loadersErrs,
		HeadEls:             headEls,
		HasRootData:         hasRootData,
	}

	return &uiRoutesData{activePathData: apd, found: true}
}
