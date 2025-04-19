import { useAtomValue } from "jotai";
import type { JSX } from "react";
import type {
	getCurrentRiverData,
	RiverRouteGeneric,
	RiverRoutePropsGeneric,
	RiverUntypedLoader,
} from "river.now/client";
import { currentRiverDataAtom, loadersDataAtom } from "./react.tsx";

export type RiverRouteProps<
	T extends RiverUntypedLoader = RiverUntypedLoader,
	Pattern extends T["pattern"] = string,
> = RiverRoutePropsGeneric<JSX.Element, T, Pattern>;

export type RiverRoute<
	T extends RiverUntypedLoader = RiverUntypedLoader,
	Pattern extends T["pattern"] = string,
> = RiverRouteGeneric<JSX.Element, T, Pattern>;

export function makeTypedUseCurrentRiverData<RD>() {
	return () =>
		useAtomValue(currentRiverDataAtom) as ReturnType<typeof getCurrentRiverData<RD>>;
}

export function makeTypedUseLoaderData<T extends RiverUntypedLoader>() {
	return function useLoaderData<P extends RiverRouteProps<T>>(
		props: P,
	): Extract<T, { pattern: P["__phantom_pattern"] }>["phantomOutputType"] | undefined {
		const loadersData = useAtomValue(loadersDataAtom);
		return loadersData?.[props.idx];
	};
}
