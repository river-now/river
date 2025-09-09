import { makeTypedNavigate } from "river.now/client";
import { addThemeChangeListener, getTheme } from "river.now/kit/theme";
import {
	makeTypedAddClientLoader,
	makeTypedLink,
	makeTypedUseLoaderData,
	makeTypedUsePatternLoaderData,
	makeTypedUseRouterData,
} from "river.now/solid";
import { createSignal } from "solid-js";
import {
	riverAppConfig,
	type RiverApp,
	type RouteProps,
} from "../river.gen.ts";

export type { RouteProps };
export const useRouterData = makeTypedUseRouterData<RiverApp>();
export const useLoaderData = makeTypedUseLoaderData<RiverApp>();
export const usePatternLoaderData = makeTypedUsePatternLoaderData<RiverApp>();
export const addClientLoader = makeTypedAddClientLoader<RiverApp>();
export const navigate = makeTypedNavigate(riverAppConfig);
export const Link = makeTypedLink(riverAppConfig, {
	prefetch: "intent",
});

/////////////////////////////////////////////////////////////////////
/////// THEME
/////////////////////////////////////////////////////////////////////

const [theme, set_theme_signal] = createSignal(getTheme());
addThemeChangeListener((e) => set_theme_signal(e.detail.theme));
export { theme };
