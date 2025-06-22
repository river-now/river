package framework

import (
	"net/http"

	"github.com/river-now/river/kit/matcher"
	"github.com/river-now/river/kit/mux"
	"github.com/river-now/river/kit/rpc"
	"github.com/river-now/river/kit/tsgen"
)

type AdHocType = rpc.AdHocType

type TSGenOptions struct {
	LoadersRouter *mux.NestedRouter
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

	allLoaders := opts.LoadersRouter.AllRoutes()
	allActions := opts.ActionsRouter.AllRoutes()

	loadersDynamicRune := opts.LoadersRouter.GetDynamicParamPrefixRune()
	actionsDynamicRune := opts.ActionsRouter.GetDynamicParamPrefixRune()

	expectedRootDataPattern := ""
	if opts.LoadersRouter.GetExplicitIndexSegment() != "" {
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
		params := extractDynamicParamsFromPattern(pattern, loadersDynamicRune)
		item.ArbitraryProperties["params"] = params
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
		if _, ok := seen[path.OriginalPattern]; ok {
			continue
		}
		item := tsgen.CollectionItem{
			ArbitraryProperties: map[string]any{
				base.DiscriminatorStr:     path.OriginalPattern,
				base.CategoryPropertyName: "loader",
			},
			PhantomTypes: map[string]AdHocType{
				"phantomOutputType": {TypeInstance: mux.None{}},
			},
		}
		params := extractDynamicParamsFromPattern(path.OriginalPattern, actionsDynamicRune)
		item.ArbitraryProperties["params"] = params
		collection = append(collection, item)
		seen[path.OriginalPattern] = struct{}{}
	}

	for _, action := range allActions {
		method, pattern := action.Method(), action.OriginalPattern()
		_, isQuery := queryMethods[method]
		_, isMutation := mutationMethods[method]
		if !isQuery && !isMutation {
			continue
		}
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
		if isMutation && method != http.MethodPost {
			item.ArbitraryProperties["method"] = method
		}
		params := extractDynamicParamsFromPattern(pattern, actionsDynamicRune)
		item.ArbitraryProperties["params"] = params
		if action != nil {
			item.PhantomTypes = map[string]AdHocType{
				"phantomInputType":  {TypeInstance: action.I()},
				"phantomOutputType": {TypeInstance: action.O()},
			}
		}
		collection = append(collection, item)
	}

	categories := []rpc.CategorySpecificOptions{}
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
	categories = append(categories, rpc.CategorySpecificOptions{
		BaseOptions:          base,
		CategoryValue:        "query",
		ItemTypeNameSingular: "RiverQuery",
		ItemTypeNamePlural:   "RiverQueries",
		KeyUnionTypeName:     "RiverQueryPattern",
		InputUnionTypeName:   "RiverQueryInput",
		OutputUnionTypeName:  "RiverQueryOutput",
	})
	categories = append(categories, rpc.CategorySpecificOptions{
		BaseOptions:          base,
		CategoryValue:        "mutation",
		ItemTypeNameSingular: "RiverMutation",
		ItemTypeNamePlural:   "RiverMutations",
		KeyUnionTypeName:     "RiverMutationPattern",
		InputUnionTypeName:   "RiverMutationInput",
		OutputUnionTypeName:  "RiverMutationOutput",
	})

	extraTSToUse := rpc.BuildFromCategories(categories)

	extraTSToUse += `export type RiverMutationMethod<T extends RiverMutationPattern> = Extract<
	RiverMutation,
	{ pattern: T }
> extends { method: infer M }
	? M extends string
		? M
		: "POST"
	: "POST";

import type { SharedBase } from "river.now/client";

export type BaseQueryProps<P extends RiverQueryPattern> = SharedBase<P>;
export type BaseMutationProps<P extends RiverMutationPattern> = SharedBase<P> &
	(RiverMutationMethod<P> extends "POST"
		? { method?: "POST" }
		: { method: RiverMutationMethod<P> });
export type BaseQueryPropsWithInput<P extends RiverQueryPattern> = BaseQueryProps<P> & {
	input: RiverQueryInput<P>;
};
export type BaseMutationPropsWithInput<P extends RiverMutationPattern> = BaseMutationProps<P> & {
	input: RiverMutationInput<P>;
};
`

	if foundRootData {
		extraTSToUse += "\nexport type RiverRootData = Extract<(typeof routes)[number], { isRootData: true }>[\"phantomOutputType\"];\n"
	} else {
		extraTSToUse += "export type RiverRootData = null;\n"
	}

	fTypeIn := []string{"RiverLoader", "RiverQuery", "RiverMutation"}
	pTypeIn := []string{"RiverLoaderPattern", "RiverQueryPattern", "RiverMutationPattern"}
	extraTSToUse += "type RiverFunction = " + tsgen.TypeUnion(fTypeIn) + ";\n"
	extraTSToUse += "type RiverPattern = " + tsgen.TypeUnion(pTypeIn) + ";\n"
	extraTSToUse += `export type RiverRouteParams<T extends RiverPattern> = (Extract<RiverFunction, { pattern: T }>["params"])[number];` + "\n"

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

func extractDynamicParamsFromPattern(pattern string, dynamicRune rune) []string {
	dynamicParams := []string{}
	segments := matcher.ParseSegments(pattern)
	for _, segment := range segments {
		if len(segment) > 0 && segment[0] == byte(dynamicRune) {
			dynamicParams = append(dynamicParams, segment[1:])
		}
	}
	return dynamicParams
}
