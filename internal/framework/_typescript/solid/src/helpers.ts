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
import { type Accessor, createMemo } from "solid-js";
import type { JSX } from "solid-js/jsx-runtime";
import { clientLoadersData, loadersData, routerData } from "./solid.tsx";

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
	return (() => routerData) as UseRouterDataFunction<
		OuterLoader,
		RootData,
		true
	>;
}

export function makeTypedUseLoaderData<Loader extends RiverUntypedFunction>() {
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
	Loader extends RiverUntypedFunction,
>() {
	return function usePatternData<Pattern extends Loader["pattern"]>(
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
			serverDataPromise: Promise<
				ClientLoaderAwaitedServerData<RootData, LoaderData>
			>;
			signal: AbortSignal;
		}) => Promise<T>,
	) {
		registerClientLoaderPattern(p as string).catch((error) => {
			console.error("Failed to register client loader pattern:", error);
		});
		(m as any)[p] = fn;

		type Res = Awaited<ReturnType<typeof fn>>;

		const useClientLoaderData = (
			props?: RiverRouteProps<Loader, Pattern>,
		): Accessor<Res | undefined> => {
			return createMemo(() => {
				if (props) {
					return clientLoadersData()[props.idx];
				}
				const matched = routerData().matchedPatterns;
				const idx = matched.findIndex((pattern) => pattern === p);
				if (idx === -1) return undefined;
				return clientLoadersData()[idx];
			});
		};

		return useClientLoaderData as {
			(props: RiverRouteProps<Loader, Pattern>): Accessor<Res>;
			(): Accessor<Res | undefined>;
		};
	};
}
