type ImportPromise = Promise<Record<string, any>>;
type Key<T extends ImportPromise> = keyof Awaited<T>;
type RouteTuple<IP extends ImportPromise> = [IP, Key<IP>] | [IP, Key<IP>, Key<IP>];

export type RiverRoutes = {
	New: <IP extends ImportPromise>(
		pattern: string,
		importPromise: IP,
		componentKey: Key<IP>,
		errorBoundaryKey?: Key<IP>,
	) => void;

	Route: <IP extends ImportPromise>(
		importPromise: IP,
		componentKey: Key<IP>,
		errorBoundaryKey?: Key<IP>,
	) => RouteTuple<IP>;
};
