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
import { createMemo, type Accessor } from "solid-js";
import type { JSX } from "solid-js/jsx-runtime";
import { clientLoadersData, loadersData, routerData } from "./solid.tsx";

export type RiverRouteProps<
	App extends RiverAppBase = any,
	Pattern extends RiverLoaderPattern<App> = string,
> = RiverRoutePropsGeneric<JSX.Element, App, Pattern>;

export type RiverRoute<
	App extends RiverAppBase = any,
	Pattern extends RiverLoaderPattern<App> = string,
> = RiverRouteGeneric<JSX.Element, App, Pattern>;

export function makeTypedUseRouterData<App extends RiverAppBase>() {
	return (() => routerData) as UseRouterDataFunction<App, true>;
}

export function makeTypedUseLoaderData<App extends RiverAppBase>() {
	return function useLoaderData<Pattern extends RiverLoaderPattern<App>>(
		props: RiverRouteProps<App, Pattern>,
	): Accessor<RiverLoaderOutput<App, Pattern>> {
		return createMemo(() => {
			return loadersData()[props.idx];
		});
	};
}

export function makeTypedUsePatternLoaderData<App extends RiverAppBase>() {
	return function usePatternLoaderData<
		Pattern extends RiverLoaderPattern<App>,
	>(pattern: Pattern): Accessor<RiverLoaderOutput<App, Pattern> | undefined> {
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
			(props: RiverRouteProps<App, Pattern>): Accessor<Res>;
			(): Accessor<Res | undefined>;
		};
	};
}
