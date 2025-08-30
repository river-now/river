import { makeTypedNavigate } from "river.now/client";
import { addThemeChangeListener, getTheme } from "river.now/kit/theme";
import {
	makeTypedAddClientLoader,
	makeTypedLink,
	makeTypedUseLoaderData,
	makeTypedUsePatternLoaderData,
	makeTypedUseRouterData,
	type RiverRouteProps,
} from "river.now/solid";
import { createSignal } from "solid-js";
import {
	apiConfig,
	type RiverLoader,
	type RiverLoaderPattern,
	type RiverRootData,
} from "../river.gen.ts";

export type RouteProps<P extends RiverLoaderPattern> = RiverRouteProps<
	RiverLoader,
	P
>;

export const useRouterData = makeTypedUseRouterData<
	RiverLoader,
	RiverRootData
>();

export const useLoaderData = makeTypedUseLoaderData<RiverLoader>();

export const addClientLoader = makeTypedAddClientLoader<
	RiverLoader,
	RiverRootData
>();

export const usePatternLoaderData =
	makeTypedUsePatternLoaderData<RiverLoader>();

export const AppLink = makeTypedLink<RiverLoader>(apiConfig, {
	prefetch: "intent",
});

export const appNavigate = makeTypedNavigate<RiverLoader>(apiConfig);

/////////////////////////////////////////////////////////////////////
/////// THEME
/////////////////////////////////////////////////////////////////////

const [theme, set_theme_signal] = createSignal(getTheme());

addThemeChangeListener((e) => set_theme_signal(e.detail.theme));

export { theme };
