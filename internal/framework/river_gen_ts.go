package framework

import (
	"net/http"

	"github.com/sjc5/river/kit/mux"
	"github.com/sjc5/river/kit/rpc"
	"github.com/sjc5/river/kit/tsgen"
)

type AdHocType = rpc.AdHocType

type TSGenOptions struct {
	UIRouter      *mux.NestedRouter
	ActionsRouter *mux.Router
	AdHocTypes    []*AdHocType
	ExtraTSCode   string
}

var base = rpc.BaseOptions{
	CollectionVarName:    "routes",
	DiscriminatorStr:     "pattern",
	CategoryPropertyName: "_type",
}

var queryMethods = map[string]struct{}{
	http.MethodGet: {}, http.MethodHead: {},
}
var mutationMethods = map[string]struct{}{
	http.MethodPost: {}, http.MethodPut: {}, http.MethodPatch: {}, http.MethodDelete: {},
}

func (h *River) GenerateTypeScript(opts *TSGenOptions) (string, error) {
	var collection []tsgen.CollectionItem

	allLoaders := opts.UIRouter.AllRoutes()
	allActions := opts.ActionsRouter.AllRoutes()

	expectedRootDataPattern := ""
	if opts.UIRouter.GetExplicitIndexSegment() != "" {
		expectedRootDataPattern = "/"
	}

	var foundRootData bool

	var seen = map[string]struct{}{}

	for pattern, loader := range allLoaders {
		item := tsgen.CollectionItem{
			ArbitraryProperties: map[string]any{
				base.DiscriminatorStr:     pattern,
				base.CategoryPropertyName: "loader",
			},
		}
		if loader != nil {
			item.PhantomTypes = map[string]AdHocType{
				"phantomOutputType": {TypeInstance: loader.O()},
			}
		}
		if pattern == expectedRootDataPattern {
			foundRootData = true
			item.ArbitraryProperties["isRootData"] = true
		}
		collection = append(collection, item)
		seen[pattern] = struct{}{}
	}

	// add any client-defined paths that don't have loaders
	// (loaders are optional, client routes are obviously required)
	maybeExtraLoaderPaths := h._paths
	for _, path := range maybeExtraLoaderPaths {
		if _, ok := seen[path.Pattern]; ok {
			continue
		}
		item := tsgen.CollectionItem{
			ArbitraryProperties: map[string]any{
				base.DiscriminatorStr:     path.Pattern,
				base.CategoryPropertyName: "loader",
			},
			PhantomTypes: map[string]AdHocType{
				"phantomOutputType": {TypeInstance: mux.None{}},
			},
		}
		collection = append(collection, item)
		seen[path.Pattern] = struct{}{}
	}

	hasQueries, hasMutations := false, false

	for _, action := range allActions {
		method, pattern := action.Method(), action.Pattern()
		_, isQuery := queryMethods[method]
		_, isMutation := mutationMethods[method]
		if !isQuery && !isMutation {
			continue
		}
		hasQueries = hasQueries || isQuery
		hasMutations = hasMutations || isMutation
		categoryPropertyName := "query"
		if isMutation {
			categoryPropertyName = "mutation"
		}
		item := tsgen.CollectionItem{
			ArbitraryProperties: map[string]any{
				base.DiscriminatorStr:     pattern,
				base.CategoryPropertyName: categoryPropertyName,
			},
		}
		if action != nil {
			item.PhantomTypes = map[string]AdHocType{
				"phantomInputType":  {TypeInstance: action.I()},
				"phantomOutputType": {TypeInstance: action.O()},
			}
		}
		collection = append(collection, item)
	}

	categories := []rpc.CategorySpecificOptions{}
	if len(allLoaders) > 0 {
		categories = append(categories, rpc.CategorySpecificOptions{
			BaseOptions:          base,
			CategoryValue:        "loader",
			ItemTypeNameSingular: "RiverLoader",
			ItemTypeNamePlural:   "RiverLoaders",
			KeyUnionTypeName:     "RiverLoaderPattern",
			InputUnionTypeName:   "",
			OutputUnionTypeName:  "RiverLoaderOutput",
			SkipInput:            true,
		})
	}
	if hasQueries {
		categories = append(categories, rpc.CategorySpecificOptions{
			BaseOptions:          base,
			CategoryValue:        "query",
			ItemTypeNameSingular: "RiverQuery",
			ItemTypeNamePlural:   "RiverQueries",
			KeyUnionTypeName:     "RiverQueryPattern",
			InputUnionTypeName:   "RiverQueryInput",
			OutputUnionTypeName:  "RiverQueryOutput",
		})
	}
	if hasMutations {
		categories = append(categories, rpc.CategorySpecificOptions{
			BaseOptions:          base,
			CategoryValue:        "mutation",
			ItemTypeNameSingular: "RiverMutation",
			ItemTypeNamePlural:   "RiverMutations",
			KeyUnionTypeName:     "RiverMutationPattern",
			InputUnionTypeName:   "RiverMutationInput",
			OutputUnionTypeName:  "RiverMutationOutput",
		})
	}

	extraTSToUse := rpc.BuildFromCategories(categories)

	if foundRootData {
		extraTSToUse += "export type RiverRootData = Extract<(typeof routes)[number], { isRootData: true }>[\"phantomOutputType\"];\n"
	} else {
		extraTSToUse += "export type RiverRootData = null;\n"
	}

	if opts.ExtraTSCode != "" {
		extraTSToUse += "\n" + opts.ExtraTSCode
	}

	return tsgen.GenerateTSContent(tsgen.Opts{
		Collection:        collection,
		CollectionVarName: base.CollectionVarName,
		AdHocTypes:        opts.AdHocTypes,
		ExtraTSCode:       extraTSToUse,
	})
}
