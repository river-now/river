import { riverNavigate } from "../client.ts";
import {
	__resolvePath,
	type ExtractApp,
	type PermissivePatternBasedProps,
	type RiverAppBase,
	type RiverAppConfig,
	type RiverLoaderPattern,
} from "../river_app_helpers/river_app_helpers.ts";
import { __riverClientGlobal } from "../river_ctx/river_ctx.ts";

type TypedNavigateOptions<
	App extends RiverAppBase,
	Pattern extends RiverLoaderPattern<App>,
> = PermissivePatternBasedProps<App, Pattern> & {
	replace?: boolean;
	scrollToTop?: boolean;
	search?: string;
	hash?: string;
	state?: unknown;
};

export function makeTypedNavigate<C extends RiverAppConfig>(riverAppConfig: C) {
	type App = ExtractApp<C>;

	return async function typedNavigate<
		Pattern extends RiverLoaderPattern<App>,
	>(options: TypedNavigateOptions<App, Pattern>): Promise<void> {
		const {
			pattern,
			params,
			splatValues,
			replace,
			scrollToTop,
			search,
			hash,
			state,
		} = options as any;

		const href = __resolvePath({
			riverAppConfig,
			type: "loader",
			props: {
				pattern,
				...(params && { params }),
				...(splatValues && { splatValues }),
			},
		});

		return riverNavigate(href, {
			replace,
			scrollToTop,
			search,
			hash,
			state,
		});
	};
}
