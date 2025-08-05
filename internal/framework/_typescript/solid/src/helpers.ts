import {
	type getRouterData,
	internal_RiverClientGlobal,
	type RiverRouteGeneric,
	type RiverRoutePropsGeneric,
	type RiverUntypedLoader,
	type UseRouterDataFunction,
	type ParamsForPattern,
} from "river.now/client";
import { type Accessor, createMemo } from "solid-js";
import type { JSX } from "solid-js/jsx-runtime";
import { clientLoadersData, loadersData, routerData } from "./solid.tsx";

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
	return (() => routerData) as UseRouterDataFunction<
		OuterLoader,
		RootData,
		true
	>;
}

export function makeTypedUseLoaderData<Loader extends RiverUntypedLoader>() {
	return function useLoaderData<
		Props extends RiverRouteProps<Loader>,
		LoaderData = Extract<
			Loader,
			{ pattern: Props["__phantom_pattern"] }
		>["phantomOutputType"],
	>(props: Props): Accessor<LoaderData> {
		return createMemo(() => {
			return loadersData()[props.idx];
		});
	};
}

export function makeTypedUsePatternLoaderData<
	Loader extends RiverUntypedLoader,
>() {
	return function usePatternData<Pattern extends string = string>(
		pattern: Pattern,
	): Accessor<
		Extract<Loader, { pattern: Pattern }>["phantomOutputType"] | undefined
	> {
		const idx = createMemo(() => {
			const matchedPatterns = routerData().matchedPatterns;
			return matchedPatterns.findIndex((p) => p === pattern);
		});
		const loaderData = createMemo(() => {
			const index = idx();
			if (index === -1) {
				return undefined;
			}
			return loadersData()[index];
		});
		return loaderData;
	};
}

export function usePatternClientLoaderData<
	ClientLoaderData extends Accessor<any> = Accessor<any>,
>(pattern: string): Accessor<ReturnType<ClientLoaderData> | undefined> {
	const idx = createMemo(() => {
		const matchedPatterns = routerData().matchedPatterns;
		return matchedPatterns.findIndex((p) => p === pattern);
	});
	const clientLoaderData = createMemo(() => {
		const index = idx();
		if (index === -1) {
			return undefined;
		}
		return clientLoadersData()[index];
	});
	return clientLoaderData;
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
		): Accessor<Awaited<ReturnType<typeof fn>>> {
			return createMemo(() => {
				return clientLoadersData()[props.idx];
			});
		};
	};
}
