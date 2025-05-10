import { addThemeChangeListener, getTheme } from "river.now/kit/theme";
import {
	makeTypedAddClientLoader,
	makeTypedUseLoaderData,
	makeTypedUsePatternLoaderData,
	makeTypedUseRouterData,
	type RiverRouteProps,
} from "river.now/solid";
import { createSignal } from "solid-js";
import type { RiverLoader, RiverLoaderPattern, RiverRootData } from "../river.gen.ts";

export type RouteProps<P extends RiverLoaderPattern> = RiverRouteProps<RiverLoader, P>;

export const useRouterData = makeTypedUseRouterData<RiverLoader, RiverRootData>();
export const useLoaderData = makeTypedUseLoaderData<RiverLoader>();
export const addClientLoader = makeTypedAddClientLoader<RiverLoader, RiverRootData>();
export const usePatternLoaderData = makeTypedUsePatternLoaderData<RiverLoader>();

const [theme, set_theme_signal] = createSignal(getTheme());
addThemeChangeListener((e) => set_theme_signal(e.detail.theme));
export { theme };
