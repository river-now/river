import { serializeToSearchParams } from "river.now/kit/json";
import type { SubmitOptions } from "./client.ts";

export type RiverAppConfig = {
	actionsRouterMountRoot: string;
	actionsDynamicRune: string;
	actionsSplatRune: string;
	loadersDynamicRune: string;
	loadersSplatRune: string;
	loadersExplicitIndexSegment: string;
	__phantom?: any;
};

export type RiverAppBase = {
	routes: readonly any[];
	appConfig: RiverAppConfig;
	rootData: any;
};

export type ExtractApp<C extends RiverAppConfig> = C["__phantom"];

type RouteByType<App extends RiverAppBase, T extends string> = Extract<
	App["routes"][number],
	{ _type: T }
>;

type RouteByPattern<Routes, P> = Extract<Routes, { pattern: P }>;

type RiverLoader<App extends RiverAppBase> = RouteByType<App, "loader">;
type RiverQuery<App extends RiverAppBase> = RouteByType<App, "query">;
type RiverMutation<App extends RiverAppBase> = RouteByType<App, "mutation">;

// Pattern types
export type RiverLoaderPattern<App extends RiverAppBase> =
	RiverLoader<App>["pattern"];
export type RiverQueryPattern<App extends RiverAppBase> =
	RiverQuery<App>["pattern"];
export type RiverMutationPattern<App extends RiverAppBase> =
	RiverMutation<App>["pattern"];

// IO types
export type RiverLoaderOutput<
	App extends RiverAppBase,
	P extends RiverLoaderPattern<App>,
> =
	RouteByPattern<RiverLoader<App>, P> extends { phantomOutputType: infer T }
		? T
		: null | undefined;

export type RiverQueryInput<
	App extends RiverAppBase,
	P extends RiverQueryPattern<App>,
> =
	RouteByPattern<RiverQuery<App>, P> extends { phantomInputType: infer T }
		? T
		: null | undefined;

export type RiverQueryOutput<
	App extends RiverAppBase,
	P extends RiverQueryPattern<App>,
> =
	RouteByPattern<RiverQuery<App>, P> extends { phantomOutputType: infer T }
		? T
		: null | undefined;

export type RiverMutationInput<
	App extends RiverAppBase,
	P extends RiverMutationPattern<App>,
> =
	RouteByPattern<RiverMutation<App>, P> extends { phantomInputType: infer T }
		? T
		: null | undefined;

export type RiverMutationOutput<
	App extends RiverAppBase,
	P extends RiverMutationPattern<App>,
> =
	RouteByPattern<RiverMutation<App>, P> extends { phantomOutputType: infer T }
		? T
		: null | undefined;

export type RiverMutationMethod<
	App extends RiverAppBase,
	P extends RiverMutationPattern<App>,
> =
	RouteByPattern<RiverMutation<App>, P> extends { method: infer M }
		? M extends string
			? M
			: "POST"
		: "POST";

// Route metadata
type RouteMetadata<App extends RiverAppBase, P extends string> = Extract<
	App["routes"][number],
	{ pattern: P }
>;

export type GetParams<App extends RiverAppBase, P extends string> =
	RouteMetadata<App, P> extends { params: ReadonlyArray<infer Params> }
		? Params extends string
			? Params
			: never
		: never;

export type RiverRouteParams<
	App extends RiverAppBase,
	P extends RiverLoaderPattern<App>,
> = GetParams<App, P>;

export type HasParams<App extends RiverAppBase, P extends string> =
	GetParams<App, P> extends never ? false : true;

export type IsSplat<App extends RiverAppBase, P extends string> =
	RouteMetadata<App, P> extends { isSplat: true } ? true : false;

export type IsEmptyInput<T> = [T] extends [null | undefined | never]
	? true
	: false;

// Pattern-based props composition
type ConditionalParams<App extends RiverAppBase, P extends string> =
	HasParams<App, P> extends true
		? { params: { [K in GetParams<App, P>]: string } }
		: {};

type ConditionalSplat<App extends RiverAppBase, P extends string> =
	IsSplat<App, P> extends true ? { splatValues: Array<string> } : {};

export type PatternBasedProps<App extends RiverAppBase, P extends string> = {
	pattern: P;
} & ConditionalParams<App, P> &
	ConditionalSplat<App, P>;

export type RiverRoutePropsGeneric<
	JSXElement,
	App extends RiverAppBase,
	P extends RiverLoaderPattern<App>,
> = {
	idx: number;
	Outlet: (props: Record<string, any>) => JSXElement;
	__phantom_pattern: P;
} & Record<string, any>;

/////////////////////////////////////////////////////////////////////
/////// API CLIENT HELPERS
/////////////////////////////////////////////////////////////////////

type Props = PatternBasedProps<any, string> & {
	options?: SubmitOptions;
	requestInit?: RequestInit;
	input?: any;
};

type APIClientHelperOpts = {
	riverAppConfig: RiverAppConfig;
	type: "loader" | "query" | "mutation";
	props: Props;
};

export type RiverQueryProps<
	App extends RiverAppBase,
	P extends RiverQueryPattern<App>,
> = (PatternBasedProps<App, P> & {
	options?: SubmitOptions;
	requestInit?: Omit<RequestInit, "method"> & { method?: "GET" };
}) &
	(IsEmptyInput<RiverQueryInput<App, P>> extends true
		? { input?: RiverQueryInput<App, P> }
		: { input: RiverQueryInput<App, P> });

export type RiverMutationProps<
	App extends RiverAppBase,
	P extends RiverMutationPattern<App>,
> = PatternBasedProps<App, P> & {
	options?: SubmitOptions;
} & (RiverMutationMethod<App, P> extends "POST"
		? { requestInit?: Omit<RequestInit, "method"> & { method?: "POST" } }
		: {
				requestInit: RequestInit & {
					method: RiverMutationMethod<App, P>;
				};
			}) &
	(IsEmptyInput<RiverMutationInput<App, P>> extends true
		? { input?: RiverMutationInput<App, P> }
		: { input: RiverMutationInput<App, P> });

export function buildQueryURL(
	riverAppConfig: RiverAppConfig,
	props: Props,
): URL {
	return buildURL({ riverAppConfig, props, type: "query" });
}

export function buildMutationURL(
	riverAppConfig: RiverAppConfig,
	props: Props,
): URL {
	return buildURL({ riverAppConfig, props, type: "mutation" });
}

function buildURL(opts: APIClientHelperOpts): URL {
	const base_path = stripTrailingSlash(
		opts.riverAppConfig.actionsRouterMountRoot,
	);
	const resolved_path = resolvePath(opts);
	const url = new URL(base_path + resolved_path, getCurrentOrigin());

	if (opts.type === "query" && opts.props.input) {
		url.search = serializeToSearchParams(opts.props.input).toString();
	}

	return url;
}

export function resolvePath(opts: APIClientHelperOpts): string {
	const { props, riverAppConfig } = opts;
	let path = props.pattern;

	let dynamicParamPrefixRune = riverAppConfig.actionsDynamicRune;
	let splatSegmentRune = riverAppConfig.actionsSplatRune;

	if (opts.type === "loader") {
		dynamicParamPrefixRune = riverAppConfig.loadersDynamicRune;
		splatSegmentRune = riverAppConfig.loadersSplatRune;
	}

	if ("params" in props && props.params) {
		for (const [key, value] of Object.entries(props.params)) {
			path = path.replace(
				`${dynamicParamPrefixRune}${key}`,
				String(value),
			);
		}
	}

	if ("splatValues" in props && props.splatValues) {
		const splatPath = (props.splatValues as Array<string>).join("/");
		path = path.replace(splatSegmentRune, splatPath);
	}

	return path;
}

function getCurrentOrigin(): string {
	return new URL(window.location.href).origin;
}

function stripTrailingSlash(path: string): string {
	return path.endsWith("/") ? path.slice(0, -1) : path;
}
