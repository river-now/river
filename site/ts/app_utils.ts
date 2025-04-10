import { addThemeChangeListener, getTheme } from "@sjc5/river/kit/theme";
import {
	makeTypedUseCurrentRiverData,
	makeTypedUseLoaderData,
	type RiverRouteProps,
} from "@sjc5/river/solid";
import { createSignal } from "solid-js";
import type { RiverLoader, RiverLoaderPattern, RiverRootData } from "./river.gen.ts";

export type RouteProps<P extends RiverLoaderPattern> = RiverRouteProps<RiverLoader, P>;

export const useCurrentAppData = makeTypedUseCurrentRiverData<RiverRootData>();
export const useLoaderData = makeTypedUseLoaderData<RiverLoader>();

const [theme, set_theme_signal] = createSignal(getTheme());
addThemeChangeListener((e) => set_theme_signal(e.detail.theme));
export { theme };
