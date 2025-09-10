import { useMemo } from "preact/hooks";
import type { JSX } from "preact/jsx-runtime";
import {
	__registerClientLoaderPattern,
	__riverClientGlobal,
	type ClientLoaderAwaitedServerData,
	type ParamsForPattern,
	type RiverAppBase,
	type RiverLoaderOutput,
	type RiverLoaderPattern,
	type RiverRouteGeneric,
	type RiverRoutePropsGeneric,
	type UseRouterDataFunction,
} from "river.now/client";
import { clientLoadersData, loadersData, routerData } from "./preact.tsx";

export type RiverRouteProps<
	App extends RiverAppBase = any,
	Pattern extends RiverLoaderPattern<App> = string,
> = RiverRoutePropsGeneric<JSX.Element, App, Pattern>;

export type RiverRoute<
	App extends RiverAppBase = any,
	Pattern extends RiverLoaderPattern<App> = string,
> = RiverRouteGeneric<JSX.Element, App, Pattern>;

export function makeTypedUseRouterData<App extends RiverAppBase>() {
	return (() => {
		return routerData.value;
	}) as UseRouterDataFunction<App, false>;
}

export function makeTypedUseLoaderData<App extends RiverAppBase>() {
	return function useLoaderData<Pattern extends RiverLoaderPattern<App>>(
		props: RiverRouteProps<App, Pattern>,
	): RiverLoaderOutput<App, Pattern> {
		return loadersData.value[props.idx];
	};
}

export function makeTypedUsePatternLoaderData<App extends RiverAppBase>() {
	return function usePatternLoaderData<
		Pattern extends RiverLoaderPattern<App>,
	>(pattern: Pattern): RiverLoaderOutput<App, Pattern> | undefined {
		const idx = useMemo(() => {
			return routerData.value.matchedPatterns.findIndex(
				(p) => p === pattern,
			);
		}, [pattern]);

		if (idx === -1) {
			return undefined;
		}
		return loadersData.value[idx];
	};
}

export function makeTypedAddClientLoader<App extends RiverAppBase>() {
	const m = __riverClientGlobal.get("patternToWaitFnMap");
	return function addClientLoader<
		Pattern extends RiverLoaderPattern<App>,
		LoaderData extends RiverLoaderOutput<App, Pattern>,
		T = any,
	>(
		p: Pattern,
		fn: (props: {
			params: Record<ParamsForPattern<App, Pattern>, string>;
			splatValues: string[];
			serverDataPromise: Promise<
				ClientLoaderAwaitedServerData<App["rootData"], LoaderData>
			>;
			signal: AbortSignal;
		}) => Promise<T>,
	) {
		__registerClientLoaderPattern(p as string).catch((error) => {
			console.error("Failed to register client loader pattern:", error);
		});
		(m as any)[p] = fn;

		type Res = Awaited<ReturnType<typeof fn>>;

		const useClientLoaderData = (
			props?: RiverRouteProps<App, Pattern>,
		): Res | undefined => {
			const idx = useMemo(() => {
				if (props) {
					return props.idx;
				}
				const matched = routerData.value.matchedPatterns;
				return matched.findIndex((pattern) => pattern === p);
			}, [props]);

			if (idx === -1) return undefined;
			return clientLoadersData.value[idx];
		};

		return useClientLoaderData as {
			(props: RiverRouteProps<App, Pattern>): Res;
			(): Res | undefined;
		};
	};
}
