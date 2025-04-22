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
	Loader extends RiverUntypedLoader = RiverUntypedLoader,
	Pattern extends Loader["pattern"] = string,
> = RiverRoutePropsGeneric<JSX.Element, Loader, Pattern>;

export type RiverRoute<
	Loader extends RiverUntypedLoader = RiverUntypedLoader,
	Pattern extends Loader["pattern"] = string,
> = RiverRouteGeneric<JSX.Element, Loader, Pattern>;

export function makeTypedUseCurrentRiverData<RootData>() {
	return () =>
		useAtomValue(currentRiverDataAtom) as ReturnType<
			typeof getCurrentRiverData<RootData>
		>;
}

export function makeTypedUseLoaderData<Loader extends RiverUntypedLoader>() {
	return function useLoaderData<
		Props extends RiverRouteProps<Loader>,
		LoaderData =
			| Extract<Loader, { pattern: Props["__phantom_pattern"] }>["phantomOutputType"]
			| undefined,
	>(props: Props): LoaderData {
		const loadersData = useAtomValue(loadersDataAtom);
		return loadersData?.[props.idx];
	};
}
