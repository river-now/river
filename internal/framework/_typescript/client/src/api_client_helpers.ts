import { serializeToSearchParams } from "river.now/kit/json";

// __TODO -- update these helpers (and quickstart templates) to use
// options obj with dedupeKey arg

export const apiHelper = {
	toQueryOpts,
	toMutationOpts,
	buildURL,
	resolveMethod,
	resolvePath,
};

type Props = SharedBase<string> & { input?: any; method?: string };

type APIClientHelperOpts = {
	actionsRouterMountRoot: string;
	type: "query" | "mutation";
	props: Props;
};

function toQueryOpts(
	actionsRouterMountRoot: string,
	props: Props,
): APIClientHelperOpts {
	return { actionsRouterMountRoot, props, type: "query" };
}

function toMutationOpts(
	actionsRouterMountRoot: string,
	props: Props,
): APIClientHelperOpts {
	return { actionsRouterMountRoot, props, type: "mutation" };
}

function buildURL(opts: APIClientHelperOpts) {
	const url = new URL(
		stripTrailingSlash(opts.actionsRouterMountRoot) +
			resolvePath(opts.props),
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

type StrIncludesColon<T extends string> =
	T extends `${infer _pre}:${infer _post}` ? true : false;

type StrEndsInAsterisk<T extends string> = T extends `${infer _pre}*`
	? true
	: false;

type PatternIsDynamic<T extends string> =
	StrIncludesColon<T> extends true
		? true
		: StrEndsInAsterisk<T> extends true
			? true
			: false;

export type SharedBase<P extends string> = {
	pattern: P;
} & (PatternIsDynamic<P> extends true ? { path: string } : { path?: string });

function resolvePath(props: SharedBase<string>) {
	return props.path || props.pattern;
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
