import { useAtomValue } from "jotai";
import { type JSX, useMemo } from "react";
import {
	type getRouterData,
	internal_RiverClientGlobal,
	type RiverRouteGeneric,
	type RiverRoutePropsGeneric,
	type RiverUntypedLoader,
	type UseRouterDataFunction,
	type ParamsForPattern,
} from "river.now/client";
import {
	clientLoadersDataAtom,
	loadersDataAtom,
	routerDataAtom,
} from "./react.tsx";

export type RiverRouteProps<
	Loader extends RiverUntypedLoader = RiverUntypedLoader,
	Pattern extends Loader["pattern"] = string,
> = RiverRoutePropsGeneric<JSX.Element, Loader, Pattern>;

export type RiverRoute<
	Loader extends RiverUntypedLoader = RiverUntypedLoader,
	Pattern extends Loader["pattern"] = string,
> = RiverRouteGeneric<JSX.Element, Loader, Pattern>;

export function makeTypedUseRouterData<
	OuterLoader extends RiverUntypedLoader,
	RootData,
>() {
	return (() => {
		return useAtomValue(routerDataAtom);
	}) as UseRouterDataFunction<OuterLoader, RootData>;
}

export function makeTypedUseLoaderData<Loader extends RiverUntypedLoader>() {
	return function useLoaderData<
		Props extends RiverRouteProps<Loader>,
		LoaderData = Extract<
			Loader,
			{ pattern: Props["__phantom_pattern"] }
		>["phantomOutputType"],
	>(props: Props): LoaderData {
		const loadersData = useAtomValue(loadersDataAtom);
		return loadersData[props.idx];
	};
}

export function makeTypedUsePatternLoaderData<
	Loader extends RiverUntypedLoader,
>() {
	return function usePatternData<Pattern extends string = string>(
		pattern: Pattern,
	): Extract<Loader, { pattern: Pattern }>["phantomOutputType"] | undefined {
		const routerData = useAtomValue(routerDataAtom);
		const loadersData = useAtomValue(loadersDataAtom);
		const idx = useMemo(() => {
			return routerData.matchedPatterns.findIndex((p) => p === pattern);
		}, [routerData.matchedPatterns, pattern]);
		if (idx === -1) {
			return undefined;
		}
		return loadersData[idx];
	};
}

export function usePatternClientLoaderData<ClientLoaderData = any>(
	pattern: string,
): ClientLoaderData | undefined {
	const routerData = useAtomValue(routerDataAtom);
	const clientLoadersData = useAtomValue(clientLoadersDataAtom);
	const idx = useMemo(() => {
		return routerData.matchedPatterns.findIndex((p) => p === pattern);
	}, [routerData.matchedPatterns, pattern]);
	if (idx === -1) {
		return undefined;
	}
	return clientLoadersData[idx];
}

export function makeTypedAddClientLoader<
	OuterLoader extends RiverUntypedLoader,
	RootData,
>() {
	const m = internal_RiverClientGlobal.get("patternToWaitFnMap");
	return function addClientLoader<
		Pattern extends OuterLoader["pattern"],
		Loader extends Extract<OuterLoader, { pattern: Pattern }>,
		RouterData = ReturnType<
			typeof getRouterData<
				RootData,
				Record<ParamsForPattern<OuterLoader, Pattern>, string>
			>
		>,
		LoaderData = Loader["phantomOutputType"],
		T = any,
	>(
		p: Pattern,
		fn: (props: RouterData & { loaderData: LoaderData }) => Promise<T>,
	) {
		(m as any)[p] = fn;

		return function useClientLoaderData(
			props: RiverRouteProps<Loader>,
		): Awaited<ReturnType<typeof fn>> {
			const clientLoadersData = useAtomValue(clientLoadersDataAtom);
			return clientLoadersData[props.idx];
		};
	};
}
