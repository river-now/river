import { addThemeChangeListener, getTheme } from "river.now/kit/theme";
import {
	makeTypedAddClientLoader,
	makeTypedUseCurrentRiverData,
	makeTypedUseLoaderData,
	type RiverRouteProps,
} from "river.now/solid";
import { createSignal } from "solid-js";
import type { RiverLoader, RiverLoaderPattern, RiverRootData } from "../river.gen.ts";

export type RouteProps<P extends RiverLoaderPattern> = RiverRouteProps<RiverLoader, P>;

export const useCurrentAppData = makeTypedUseCurrentRiverData<RiverRootData>();
export const useLoaderData = makeTypedUseLoaderData<RiverLoader>();
export const addClientLoader = makeTypedAddClientLoader<RiverLoader, RiverRootData>();

const [theme, set_theme_signal] = createSignal(getTheme());
addThemeChangeListener((e) => set_theme_signal(e.detail.theme));
export { theme };
