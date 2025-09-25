/// <reference types="vite/client" />

import { useMemo, type JSX } from "react";
import {
	__registerClientLoaderPattern,
	__riverClientGlobal,
	__runClientLoadersAfterHMRUpdate,
	type ClientLoaderAwaitedServerData,
	type ParamsForPattern,
	type RiverAppBase,
	type RiverLoaderOutput,
	type RiverLoaderPattern,
	type RiverRouteGeneric,
	type RiverRoutePropsGeneric,
	type UseRouterDataFunction,
} from "river.now/client";
import {
	useClientLoadersData,
	useLoadersData,
	useRouterData,
} from "./react.tsx";

export type RiverRouteProps<
	App extends RiverAppBase = any,
	Pattern extends RiverLoaderPattern<App> = string,
> = RiverRoutePropsGeneric<JSX.Element, App, Pattern>;

export type RiverRoute<
	App extends RiverAppBase = any,
	Pattern extends RiverLoaderPattern<App> = string,
> = RiverRouteGeneric<JSX.Element, App, Pattern>;

export function makeTypedUseRouterData<App extends RiverAppBase>() {
	return useRouterData as UseRouterDataFunction<App, false>;
}

export function makeTypedUseLoaderData<App extends RiverAppBase>() {
	return function useLoaderData<Pattern extends RiverLoaderPattern<App>>(
		props: RiverRouteProps<App, Pattern>,
	): RiverLoaderOutput<App, Pattern> {
		const loadersData = useLoadersData();
		return loadersData[props.idx];
	};
}

export function makeTypedUsePatternLoaderData<App extends RiverAppBase>() {
	return function usePatternLoaderData<
		Pattern extends RiverLoaderPattern<App>,
	>(pattern: Pattern): RiverLoaderOutput<App, Pattern> | undefined {
		const routerData = useRouterData();
		const loadersData = useLoadersData();
		const idx = useMemo(() => {
			return routerData.matchedPatterns.findIndex((p) => p === pattern);
		}, [routerData.matchedPatterns, pattern]);

		if (idx === -1) {
			return undefined;
		}
		return loadersData[idx];
	};
}

export function makeTypedAddClientLoader<App extends RiverAppBase>() {
	const m = __riverClientGlobal.get("patternToWaitFnMap");
	return function addClientLoader<
		Pattern extends RiverLoaderPattern<App>,
		LoaderData extends RiverLoaderOutput<App, Pattern>,
		T = any,
	>(props: {
		pattern: Pattern;
		clientLoader: (props: {
			params: Record<ParamsForPattern<App, Pattern>, string>;
			splatValues: string[];
			serverDataPromise: Promise<
				ClientLoaderAwaitedServerData<App["rootData"], LoaderData>
			>;
			signal: AbortSignal;
		}) => Promise<T>;
		reRunOnModuleChange?: ImportMeta;
	}) {
		const p = props.pattern;
		const fn = props.clientLoader;

		__registerClientLoaderPattern(p as string).catch((error) => {
			console.error("Failed to register client loader pattern:", error);
		});
		(m as any)[p] = fn;

		if (import.meta.env.DEV && props.reRunOnModuleChange) {
			__runClientLoadersAfterHMRUpdate(props.reRunOnModuleChange, p);
		}

		type Res = Awaited<ReturnType<typeof fn>>;

		const useClientLoaderData = (
			props?: RiverRouteProps<App, Pattern>,
		): Res | undefined => {
			const clientLoadersData = useClientLoadersData();
			const routerData = useRouterData();

			const idx = useMemo(() => {
				if (props) {
					return props.idx;
				}
				const matched = routerData.matchedPatterns;
				return matched.findIndex((pattern) => pattern === p);
			}, [props, routerData.matchedPatterns]);

			if (idx === -1) return undefined;
			return clientLoadersData[idx];
		};

		return useClientLoaderData as {
			(props: RiverRouteProps<App, Pattern>): Res;
			(): Res | undefined;
		};
	};
}
