export type HeadEl = {
	tag?: string;
	attributesKnownSafe?: Record<string, string>;
	booleanAttributes?: Array<string>;
	dangerousInnerHTML?: string;
};

type Meta = {
	title: string;
	metaHeadEls: Array<HeadEl>;
	restHeadEls: Array<HeadEl>;
};

type shared = {
	loadersData: Array<any>;
	importURLs: Array<string>;
	exportKeys: Array<string>;
	outermostErrorIndex: number;
	params: Record<string, string>;
	matchedPatterns: Array<string>;
	splatValues: Array<string>;
	hasRootData: boolean;
	buildID: string;
	activeErrorBoundaries: Array<any> | null;
	activeComponents: Array<any> | null;
};

export type GetRouteDataOutput = Omit<shared, "buildID"> &
	Meta & {
		deps: Array<string>;
		cssBundles: Array<string>;
	};

export const RIVER_SYMBOL = Symbol.for("__river_internal__");

export type RiverClientGlobal = shared & {
	isDev: boolean;
	viteDevURL: string;
	publicPathPrefix: string;
	isTouchDevice: boolean;
	patternToWaitFnMap: Record<
		string,
		(
			props: ReturnType<typeof getCurrentRiverData> & { loaderData: any },
		) => Promise<any>
	>;
	clientLoadersData: Array<any>;
};

export function __getRiverClientGlobal() {
	const dangerousGlobalThis = globalThis as any;
	function get<K extends keyof RiverClientGlobal>(key: K) {
		return dangerousGlobalThis[RIVER_SYMBOL][key] as RiverClientGlobal[K];
	}
	function set<K extends keyof RiverClientGlobal, V extends RiverClientGlobal[K]>(
		key: K,
		value: V,
	) {
		dangerousGlobalThis[RIVER_SYMBOL][key] = value;
	}
	return { get, set };
}

export const internal_RiverClientGlobal = __getRiverClientGlobal();

// to debug ctx in browser, paste this:
// const river_ctx = window[Symbol.for("__river_internal__")];

export function getCurrentRiverData<T = any>() {
	let rootData: T | null = null;
	if (internal_RiverClientGlobal.get("hasRootData")) {
		rootData = internal_RiverClientGlobal.get("loadersData")[0];
	}
	return {
		buildID: internal_RiverClientGlobal.get("buildID") || "",
		matchedPatterns: internal_RiverClientGlobal.get("matchedPatterns") || [],
		splatValues: internal_RiverClientGlobal.get("splatValues") || [],
		params: internal_RiverClientGlobal.get("params") || {},
		rootData,
	};
}
