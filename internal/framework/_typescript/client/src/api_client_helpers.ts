import { serializeToSearchParams } from "river.now/kit/json";
import type { SubmitOptions } from "./client.ts";
import type {
	PatternBasedProps,
	RiverAppBase,
	RiverMutationInput,
	RiverMutationMethod,
	RiverMutationPattern,
	RiverQueryInput,
	RiverQueryPattern,
	WithOptionalInput,
} from "./river_app_types.ts";

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

export type RiverAppConfig = {
	actionsRouterMountRoot: string;
	actionsDynamicRune: string;
	actionsSplatRune: string;
	loadersDynamicRune: string;
	loadersSplatRune: string;
	loadersExplicitIndexSegment: string;
	__phantom?: any;
};

export type QueryProps<
	App extends RiverAppBase,
	P extends RiverQueryPattern<App>,
> = {
	pattern: P;
	options?: SubmitOptions;
	requestInit?: Omit<RequestInit, "method"> & { method?: "GET" };
} & PatternBasedProps<App, P> &
	WithOptionalInput<RiverQueryInput<App, P>>;

export type MutationProps<
	App extends RiverAppBase,
	P extends RiverMutationPattern<App>,
> = {
	pattern: P;
	options?: SubmitOptions;
} & PatternBasedProps<App, P> &
	(RiverMutationMethod<App, P> extends "POST"
		? { requestInit?: Omit<RequestInit, "method"> & { method?: "POST" } }
		: {
				requestInit: RequestInit & {
					method: RiverMutationMethod<App, P>;
				};
			}) &
	WithOptionalInput<RiverMutationInput<App, P>>;

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

function buildURL(opts: APIClientHelperOpts) {
	const url = new URL(
		stripTrailingSlash(opts.riverAppConfig.actionsRouterMountRoot) +
			resolvePath(opts),
		getCurrentOrigin(),
	);
	if (opts.type === "query" && opts.props.input) {
		url.search = serializeToSearchParams(opts.props.input).toString();
	}
	return url;
}

export function resolvePath(opts: APIClientHelperOpts) {
	const { props, riverAppConfig } = opts;
	let path = props.pattern;

	let dynamicParamPrefixRune = riverAppConfig.actionsDynamicRune;
	let splatSegmentRune = riverAppConfig.actionsSplatRune;

	if (opts.type === "loader") {
		dynamicParamPrefixRune = riverAppConfig.loadersDynamicRune;
		splatSegmentRune = riverAppConfig.loadersSplatRune;
	}

	// Replace parameter placeholders with actual values
	if ("params" in props && props.params) {
		for (const [key, value] of Object.entries(props.params)) {
			path = path.replace(`${dynamicParamPrefixRune}${key}`, value);
		}
	}

	// Replace splat marker with splat values
	if ("splatValues" in props && props.splatValues) {
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
