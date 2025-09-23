// Used by client route defs file (e.g., river.routes.ts)

type ImportPromise = Promise<Record<string, any>>;
type Key<T extends ImportPromise> = keyof Awaited<T>;

export type RiverRoutes = {
	Add: <IP extends ImportPromise>(
		pattern: string,
		importPromise: IP,
		componentKey: Key<IP>,
		errorBoundaryKey?: Key<IP>,
	) => void;
};
