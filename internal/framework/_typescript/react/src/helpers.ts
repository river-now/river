import { useAtomValue } from "jotai";
import { type JSX, useMemo } from "react";
import {
	type ClientLoaderAwaitedServerData,
	internal_RiverClientGlobal,
	type ParamsForPattern,
	registerClientLoaderPattern,
	type RiverRouteGeneric,
	type RiverRoutePropsGeneric,
	type RiverUntypedFunction,
	type UseRouterDataFunction,
} from "river.now/client";
import {
	clientLoadersDataAtom,
	loadersDataAtom,
	routerDataAtom,
} from "./react.tsx";

export type RiverRouteProps<
	Loader extends RiverUntypedFunction = RiverUntypedFunction,
	Pattern extends Loader["pattern"] = string,
> = RiverRoutePropsGeneric<JSX.Element, Loader, Pattern>;

export type RiverRoute<
	Loader extends RiverUntypedFunction = RiverUntypedFunction,
	Pattern extends Loader["pattern"] = string,
> = RiverRouteGeneric<JSX.Element, Loader, Pattern>;

export function makeTypedUseRouterData<
	OuterLoader extends RiverUntypedFunction,
	RootData,
>() {
	return (() => {
		return useAtomValue(routerDataAtom);
	}) as UseRouterDataFunction<OuterLoader, RootData>;
}

export function makeTypedUseLoaderData<Loader extends RiverUntypedFunction>() {
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
	Loader extends RiverUntypedFunction,
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
	OuterLoader extends RiverUntypedFunction,
	RootData,
>() {
	const m = internal_RiverClientGlobal.get("patternToWaitFnMap");
	return function addClientLoader<
		Pattern extends OuterLoader["pattern"],
		Loader extends Extract<OuterLoader, { pattern: Pattern }>,
		LoaderData = Loader["phantomOutputType"],
		T = any,
	>(
		p: Pattern,
		fn: (props: {
			params: Record<ParamsForPattern<OuterLoader, Pattern>, string>;
			splatValues: string[];
			serverDataPromise: ClientLoaderAwaitedServerData<
				RootData,
				LoaderData
			>;
			signal: AbortSignal;
		}) => Promise<T>,
	) {
		registerClientLoaderPattern(p as string).catch((error) => {
			console.error("Failed to register client loader pattern:", error);
		});
		(m as any)[p] = fn;

		return function useClientLoaderData(
			props: RiverRouteProps<Loader>,
		): Awaited<ReturnType<typeof fn>> {
			const clientLoadersData = useAtomValue(clientLoadersDataAtom);
			return clientLoadersData[props.idx];
		};
	};
}
