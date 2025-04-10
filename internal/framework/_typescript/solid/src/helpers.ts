import {
	addRouteChangeListener,
	getCurrentRiverData,
	type RiverRouteGeneric,
	type RiverRoutePropsGeneric,
	type RiverUntypedLoader,
} from "@sjc5/river/client";
import { createSignal } from "solid-js";
import type { JSX } from "solid-js/jsx-runtime";
import { loadersData } from "./solid.tsx";

export type RiverRouteProps<
	T extends RiverUntypedLoader = RiverUntypedLoader,
	Pattern extends T["pattern"] = string,
> = RiverRoutePropsGeneric<JSX.Element, T, Pattern>;

export type RiverRoute<
	T extends RiverUntypedLoader = RiverUntypedLoader,
	Pattern extends T["pattern"] = string,
> = RiverRouteGeneric<JSX.Element, T, Pattern>;

export function makeTypedUseCurrentRiverData<RD>() {
	const [currentRiverData, setCurrentRiverData] = createSignal(getCurrentRiverData<RD>());
	addRouteChangeListener(() => setCurrentRiverData(getCurrentRiverData<RD>()));
	return currentRiverData;
}

export function makeTypedUseLoaderData<T extends RiverUntypedLoader>() {
	return function useLoaderData<P extends RiverRouteProps<T>>(
		props: P,
	): Extract<T, { pattern: P["__phantom_pattern"] }>["phantomOutputType"] | undefined {
		return loadersData()?.[props.idx];
	};
}
