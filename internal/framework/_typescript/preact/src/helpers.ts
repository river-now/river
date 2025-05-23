import { useMemo } from "preact/hooks";
import type { JSX } from "preact/jsx-runtime";
import {
	type getRouterData,
	internal_RiverClientGlobal,
	type RiverRouteGeneric,
	type RiverRoutePropsGeneric,
	type RiverUntypedLoader,
	type UseRouterDataFunction,
} from "river.now/client";
import { clientLoadersData, loadersData, routerData } from "./preact.tsx";

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
		return routerData.value;
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
		return loadersData.value[props.idx];
	};
}

export function makeTypedUsePatternLoaderData<Loader extends RiverUntypedLoader>() {
	return function usePatternData<Pattern extends string = string>(
		pattern: Pattern,
	): Extract<Loader, { pattern: Pattern }>["phantomOutputType"] | undefined {
		const idx = useMemo(() => {
			return routerData.value.matchedPatterns.findIndex((p) => p === pattern);
		}, [pattern]);

		if (idx === -1) {
			return undefined;
		}
		return loadersData.value[idx];
	};
}

export function usePatternClientLoaderData<ClientLoaderData = any>(
	pattern: string,
): ClientLoaderData | undefined {
	const idx = useMemo(() => {
		return routerData.value.matchedPatterns.findIndex((p) => p === pattern);
	}, [pattern]);

	if (idx === -1) {
		return undefined;
	}
	return clientLoadersData.value[idx];
}

export function makeTypedAddClientLoader<
	OuterLoader extends RiverUntypedLoader,
	RootData,
>() {
	const m = internal_RiverClientGlobal.get("patternToWaitFnMap");

	return function addClientLoader<
		Pattern extends OuterLoader["pattern"],
		Loader extends Extract<OuterLoader, { pattern: Pattern }>,
		RouterData = ReturnType<typeof getRouterData<RootData>>,
		LoaderData = Loader["phantomOutputType"],
		T = any,
	>(p: Pattern, fn: (props: RouterData & { loaderData: LoaderData }) => Promise<T>) {
		(m as any)[p] = fn;

		return function useClientLoaderData(
			props: RiverRouteProps<Loader>,
		): Awaited<ReturnType<typeof fn>> {
			return clientLoadersData.value[props.idx];
		};
	};
}
