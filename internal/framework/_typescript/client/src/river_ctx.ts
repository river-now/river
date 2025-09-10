import type { RiverAppConfig } from "./river_app_helpers.ts";

export type HeadEl = {
	tag?: string;
	attributesKnownSafe?: Record<string, string>;
	booleanAttributes?: Array<string>;
	dangerousInnerHTML?: string;
};

type Meta = {
	title: HeadEl | null | undefined;
	metaHeadEls: Array<HeadEl> | null | undefined;
	restHeadEls: Array<HeadEl> | null | undefined;
};

type shared = {
	outermostError?: string;
	outermostErrorIdx?: number;
	errorExportKey?: string;

	matchedPatterns: Array<string>;
	loadersData: Array<any>;
	importURLs: Array<string>;
	exportKeys: Array<string>;
	hasRootData: boolean;

	params: Record<string, string>;
	splatValues: Array<string>;

	buildID: string;

	activeComponents: Array<any> | null;
	activeErrorBoundary?: any;
};

export type GetRouteDataOutput = Omit<shared, "buildID"> &
	Meta & {
		deps: Array<string>;
		cssBundles: Array<string>;
	};

export const RIVER_SYMBOL = Symbol.for("__river_internal__");

export type RouteErrorComponent = (props: { error: string }) => any;

export type ClientLoaderAwaitedServerData<RD, LD> = {
	matchedPatterns: string[];
	loaderData: LD;
	rootData: RD;
	buildID: string;
};

export type RiverClientGlobal = shared & {
	isDev: boolean;
	viteDevURL: string;
	publicPathPrefix: string;
	isTouchDevice: boolean;
	patternToWaitFnMap: Record<
		string,
		(props: {
			params: Record<string, string>;
			splatValues: string[];
			serverDataPromise: Promise<ClientLoaderAwaitedServerData<any, any>>;
			signal: AbortSignal;
		}) => Promise<any>
	>;
	clientLoadersData: Array<any>;
	defaultErrorBoundary: RouteErrorComponent;
	useViewTransitions: boolean;
	deploymentID: string;
	riverAppConfig: RiverAppConfig;
};

export function __getRiverClientGlobal() {
	const dangerousGlobalThis = globalThis as any;
	function get<K extends keyof RiverClientGlobal>(key: K) {
		return dangerousGlobalThis[RIVER_SYMBOL][key] as RiverClientGlobal[K];
	}
	function set<
		K extends keyof RiverClientGlobal,
		V extends RiverClientGlobal[K],
	>(key: K, value: V) {
		dangerousGlobalThis[RIVER_SYMBOL][key] = value;
	}
	return { get, set };
}

export const __riverClientGlobal = __getRiverClientGlobal();

// to debug ctx in browser, paste this:
// const river_ctx = window[Symbol.for("__river_internal__")];

export function getRouterData<
	T = any,
	P extends Record<string, string> = Record<string, string>,
>() {
	const rootData: T = __riverClientGlobal.get("hasRootData")
		? __riverClientGlobal.get("loadersData")[0]
		: null;
	return {
		buildID: __riverClientGlobal.get("buildID") || "",
		matchedPatterns: __riverClientGlobal.get("matchedPatterns") || [],
		splatValues: __riverClientGlobal.get("splatValues") || [],
		params: (__riverClientGlobal.get("params") || {}) as P,
		rootData,
	};
}
