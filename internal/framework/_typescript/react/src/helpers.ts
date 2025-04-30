import { useAtomValue } from "jotai";
import type { JSX } from "react";
import {
	type getCurrentRiverData,
	internal_RiverClientGlobal,
	type RiverRouteGeneric,
	type RiverRoutePropsGeneric,
	type RiverUntypedLoader,
} from "river.now/client";
import {
	clientLoadersDataAtom,
	currentRiverDataAtom,
	loadersDataAtom,
} from "./react.tsx";

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

export function makeTypedAddClientLoader<
	OuterLoader extends RiverUntypedLoader,
	RootData,
>() {
	const m = internal_RiverClientGlobal.get("patternToWaitFnMap");
	return function addClientLoader<
		Pattern extends OuterLoader["pattern"],
		Loader extends Extract<OuterLoader, { pattern: Pattern }>,
		CurrentRiverData = ReturnType<typeof getCurrentRiverData<RootData>>,
		LoaderData = Loader["phantomOutputType"] | undefined,
		T = any,
	>(
		p: Pattern,
		fn: (props: CurrentRiverData & { loaderData: LoaderData }) => Promise<T>,
	) {
		(m as any)[p] = fn;

		return function useClientLoaderData(
			props: RiverRouteProps<Loader>,
		): Awaited<ReturnType<typeof fn>> | undefined {
			const clientLoadersData = useAtomValue(clientLoadersDataAtom);
			return clientLoadersData?.[props.idx];
		};
	};
}
