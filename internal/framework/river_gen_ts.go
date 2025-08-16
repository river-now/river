package framework

import (
	"fmt"
	"net/http"
	"strings"

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

const tsTemplate = `
import type { SharedBase, WithOptionalInput } from "river.now/client";

const ACTIONS_ROUTER_MOUNT_ROOT = "%s";
const RIVER_LOADERS_DYNAMIC_RUNE = "%s";
const RIVER_LOADERS_SPLAT_RUNE = "%s";
const RIVER_ACTIONS_DYNAMIC_RUNE = "%s";
const RIVER_ACTIONS_SPLAT_RUNE = "%s";

export const apiConfig = {
	actionsRouterMountRoot: ACTIONS_ROUTER_MOUNT_ROOT,
	actionsDynamicRune: RIVER_ACTIONS_DYNAMIC_RUNE,
	actionsSplatRune: RIVER_ACTIONS_SPLAT_RUNE,
	loadersDynamicRune: RIVER_LOADERS_DYNAMIC_RUNE,
	loadersSplatRune: RIVER_LOADERS_SPLAT_RUNE,
} as const;

export type RiverMutationMethod<T extends RiverMutationPattern> =
	Extract<RiverMutation, { pattern: T }> extends { method: infer M }
		? M extends string
			? M
			: "POST"
		: "POST";

export type BaseQueryProps<P extends RiverQueryPattern> = SharedBase<
	P,
	RiverFunction
>;

export type BaseMutationProps<P extends RiverMutationPattern> = SharedBase<
	P,
	RiverFunction
> &
	(RiverMutationMethod<P> extends "POST"
		? { method?: "POST" }
		: { method: RiverMutationMethod<P> });

export type BaseQueryPropsWithInput<P extends RiverQueryPattern> =
	BaseQueryProps<P> & WithOptionalInput<RiverQueryInput<P>>;

export type BaseMutationPropsWithInput<P extends RiverMutationPattern> =
	BaseMutationProps<P> & WithOptionalInput<RiverMutationInput<P>>;
`

func (h *River) GenerateTypeScript(opts *TSGenOptions) (string, error) {
	var collection []tsgen.CollectionItem

	allLoaders := opts.LoadersRouter.AllRoutes()
	allActions := opts.ActionsRouter.AllRoutes()

	loadersDynamicRune := opts.LoadersRouter.GetDynamicParamPrefixRune()
	loadersSplatRune := opts.LoadersRouter.GetSplatSegmentRune()
	actionsDynamicRune := opts.ActionsRouter.GetDynamicParamPrefixRune()
	actionsSplatRune := opts.ActionsRouter.GetSplatSegmentRune()

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
		if len(params) > 0 {
			item.ArbitraryProperties["params"] = params
		}
		if isSplat(pattern, loadersSplatRune) {
			item.ArbitraryProperties["isSplat"] = true
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
		if len(params) > 0 {
			item.ArbitraryProperties["params"] = params
		}
		if isSplat(path.OriginalPattern, actionsSplatRune) {
			item.ArbitraryProperties["isSplat"] = true
		}
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
		if len(params) > 0 {
			item.ArbitraryProperties["params"] = params
		}
		if isSplat(pattern, actionsSplatRune) {
			item.ArbitraryProperties["isSplat"] = true
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

	extraTSToUse += fmt.Sprintf(tsTemplate,
		opts.ActionsRouter.MountRoot(),
		string(loadersDynamicRune),
		string(loadersSplatRune),
		string(actionsDynamicRune),
		string(actionsSplatRune),
	)

	if foundRootData {
		extraTSToUse += "\nexport type RiverRootData = Extract<(typeof routes)[number], { isRootData: true }>[\"phantomOutputType\"];\n\n"
	} else {
		extraTSToUse += "export type RiverRootData = null;\n\n"
	}

	fTypeIn := []string{"RiverLoader", "RiverQuery", "RiverMutation"}
	pTypeIn := []string{"RiverLoaderPattern", "RiverQueryPattern", "RiverMutationPattern"}
	extraTSToUse += "type RiverFunction = " + tsgen.TypeUnion(fTypeIn) + ";\n\n"
	extraTSToUse += "type RiverPattern = " + tsgen.TypeUnion(pTypeIn) + ";\n\n"
	extraTSToUse += `export type RiverRouteParams<T extends RiverPattern> =
	Extract<RiverFunction, { pattern: T }> extends { params: infer P }
		? P extends ReadonlyArray<string>
			? P[number]
			: never
		: never;` + "\n"

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

func isSplat(pattern string, splatRune rune) bool {
	return strings.HasSuffix(pattern, "/"+string(splatRune))
}
