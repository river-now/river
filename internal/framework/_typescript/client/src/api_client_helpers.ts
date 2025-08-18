import { serializeToSearchParams } from "river.now/kit/json";
import type { SubmitOptions } from "./client.ts";
import type { RiverUntypedFunction } from "./impl_helpers.ts";

export const apiHelper = {
	toQueryOpts,
	toMutationOpts,
	buildURL,
	resolveMethod,
	resolvePath,
};

type Props = SharedBase<string, RiverUntypedFunction> & {
	input?: any;
	method?: string;
};

type APIClientHelperOpts = {
	apiConfig: APIConfig;
	type: "loader" | "query" | "mutation";
	props: Props;
};

export type APIConfig = {
	actionsRouterMountRoot: string;
	actionsDynamicRune: string;
	actionsSplatRune: string;
	loadersDynamicRune: string;
	loadersSplatRune: string;
	loadersExplicitIndexSegment: string;
};

function toQueryOpts(apiConfig: APIConfig, props: Props): APIClientHelperOpts {
	return { apiConfig, props, type: "query" };
}

function toMutationOpts(
	apiConfig: APIConfig,
	props: Props,
): APIClientHelperOpts {
	return { apiConfig, props, type: "mutation" };
}

function buildURL(opts: APIClientHelperOpts) {
	const url = new URL(
		stripTrailingSlash(opts.apiConfig.actionsRouterMountRoot) +
			resolvePath(opts),
		getCurrentOrigin(),
	);
	if (opts.type === "query" && opts.props.input) {
		url.search = serializeToSearchParams(opts.props.input).toString();
	}
	return url;
}

function resolveMethod(opts: APIClientHelperOpts) {
	if (opts.type === "mutation") {
		return opts.props.method || "POST";
	}
	return "GET";
}

export type GetParams<T extends string, F extends RiverUntypedFunction> =
	Extract<F, { pattern: T }> extends { params: ReadonlyArray<infer P> }
		? P extends string
			? P
			: never
		: never;

export type HasParams<T extends string, F extends RiverUntypedFunction> =
	GetParams<T, F> extends never ? false : true;

export type IsSplat<T extends string, F extends RiverUntypedFunction> =
	Extract<F, { pattern: T }> extends { isSplat: true } ? true : false;

export type IsEmptyInput<T> = [T] extends [null | undefined | never]
	? true
	: false;

export type WithOptionalInput<TInput> =
	IsEmptyInput<TInput> extends true ? { input?: TInput } : { input: TInput };

export type PatternBasedProps<
	P extends string,
	F extends RiverUntypedFunction,
> = {
	pattern: P;
} & (HasParams<P, F> extends true
	? IsSplat<P, F> extends true
		? {
				params: { [K in GetParams<P, F>]: string };
				splatValues: Array<string>;
			}
		: {
				params: { [K in GetParams<P, F>]: string };
			}
	: IsSplat<P, F> extends true
		? {
				splatValues: Array<string>;
			}
		: {});

export type SharedBase<P extends string, F extends RiverUntypedFunction> = {
	pattern: P;
	options?: SubmitOptions;
} & PatternBasedProps<P, F>;

export function resolvePath(opts: APIClientHelperOpts) {
	const { props, apiConfig } = opts;
	let path = props.pattern;

	let dynamicParamPrefixRune = apiConfig.actionsDynamicRune;
	let splatSegmentRune = apiConfig.actionsSplatRune;

	if (opts.type === "loader") {
		dynamicParamPrefixRune = apiConfig.loadersDynamicRune;
		splatSegmentRune = apiConfig.loadersSplatRune;
	}

	// Replace parameter placeholders with actual values
	if ("params" in props && props.params) {
		for (const [key, value] of Object.entries(props.params)) {
			path = path.replace(`${dynamicParamPrefixRune}${key}`, value);
		}
	}

	// Replace splat marker with splat values
	if ("splatValues" in props) {
		const splatPath = (props.splatValues as Array<string>).join("/");
		path = path.replace(splatSegmentRune, splatPath);
	}

	return path;
}

function getCurrentOrigin() {
	return new URL(window.location.href).origin;
}

function stripTrailingSlash(path: string) {
	if (path.endsWith("/")) {
		return path.slice(0, -1);
	}
	return path;
}
