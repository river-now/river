// __TODO set up go/ts type sharing script

export type HeadBlock = {
	tag?: string;
	safeAttributes?: Record<string, string>;
	booleanAttributes?: Array<string>;
	innerHTML?: string;
};

type Meta = { title: string; metaHeadBlocks: Array<HeadBlock>; restHeadBlocks: Array<HeadBlock> };

type shared = {
	loadersData: Array<any>;
	importURLs: Array<string>;
	exportKeys: Array<string>;
	outermostErrorIndex: number;
	params: Record<string, string>;
	splatValues: Array<string>;
	coreData: any;
	buildID: string;
	activeErrorBoundaries: Array<any> | null;
	activeComponents: Array<any> | null;
};

export type GetRouteDataOutput = shared &
	Meta & {
		deps: Array<string>;
		cssBundles: Array<string>;
	};

export const RIVER_SYMBOL = Symbol.for("__river_internal__");

export type RiverClientGlobal = shared & {
	isDev: boolean;
	viteDevURL: string;
	publicPathPrefix: string;
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
