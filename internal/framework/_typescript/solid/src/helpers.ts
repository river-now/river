import type {
	getCurrentRiverData,
	RiverRouteGeneric,
	RiverRoutePropsGeneric,
	RiverUntypedLoader,
} from "river.now/client";
import type { Accessor } from "solid-js";
import type { JSX } from "solid-js/jsx-runtime";
import { currentRiverData, loadersData } from "./solid.tsx";

export type RiverRouteProps<
	Loader extends RiverUntypedLoader = RiverUntypedLoader,
	Pattern extends Loader["pattern"] = string,
> = RiverRoutePropsGeneric<JSX.Element, Loader, Pattern>;

export type RiverRoute<
	Loader extends RiverUntypedLoader = RiverUntypedLoader,
	Pattern extends Loader["pattern"] = string,
> = RiverRouteGeneric<JSX.Element, Loader, Pattern>;

export function makeTypedUseCurrentRiverData<RootData>() {
	return currentRiverData as Accessor<
		ReturnType<typeof getCurrentRiverData<RootData>>
	>;
}

export function makeTypedUseLoaderData<Loader extends RiverUntypedLoader>() {
	return function useLoaderData<
		Props extends RiverRouteProps<Loader>,
		LoaderData =
			| Extract<Loader, { pattern: Props["__phantom_pattern"] }>["phantomOutputType"]
			| undefined,
	>(props: Props): LoaderData {
		return loadersData()?.[props.idx];
	};
}
