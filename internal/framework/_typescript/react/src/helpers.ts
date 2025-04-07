import type { getCurrentRiverData, Route, RouteProps, UntypedLoader } from "@sjc5/river/client";
import { useAtomValue } from "jotai";
import type { JSX } from "react";
import { currentRiverDataAtom, loadersDataAtom } from "./react.tsx";

export type RiverRouteProps<
	T extends UntypedLoader = UntypedLoader,
	Pattern extends T["pattern"] = string,
> = RouteProps<JSX.Element, T, Pattern>;

export type RiverRoute<
	T extends UntypedLoader = UntypedLoader,
	Pattern extends T["pattern"] = string,
> = Route<JSX.Element, T, Pattern>;

export function makeTypedUseCurrentRiverData<RD>() {
	return () => useAtomValue(currentRiverDataAtom) as ReturnType<typeof getCurrentRiverData<RD>>;
}

export function makeTypedUseLoaderData<T extends UntypedLoader>() {
	return function useLoaderData<P extends RiverRouteProps<T>>(
		props: P,
	): Extract<T, { pattern: P["__phantom_pattern"] }>["phantomOutputType"] | undefined {
		const loadersData = useAtomValue(loadersDataAtom);
		return loadersData?.[props.idx];
	};
}
