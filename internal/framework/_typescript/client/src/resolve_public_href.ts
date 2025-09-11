import { __riverClientGlobal } from "./river_ctx/river_ctx.ts";

export function resolvePublicHref(relativeHref: string): string {
	let baseURL = __riverClientGlobal.get("viteDevURL");
	if (!baseURL) {
		baseURL = __riverClientGlobal.get("publicPathPrefix");
	}
	if (baseURL.endsWith("/")) {
		baseURL = baseURL.slice(0, -1);
	}
	let final = relativeHref.startsWith("/")
		? baseURL + relativeHref
		: baseURL + "/" + relativeHref;
	return final;
}
