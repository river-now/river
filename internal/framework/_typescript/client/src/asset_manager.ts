import { resolvePublicHref } from "./resolve_public_href.ts";
import { __riverClientGlobal } from "./river_ctx/river_ctx.ts";

export class AssetManager {
	static preloadModule(url: string): void {
		const href = resolvePublicHref(url);
		if (document.querySelector(`link[href="${CSS.escape(href)}"]`)) {
			return;
		}

		const link = document.createElement("link");
		link.rel = "modulepreload";
		link.href = href;
		document.head.appendChild(link);
	}

	static preloadCSS(url: string): Promise<void> {
		const href = resolvePublicHref(url);

		const link = document.createElement("link");
		link.rel = "preload";
		link.setAttribute("as", "style");
		link.href = href;

		document.head.appendChild(link);

		return new Promise((resolve, reject) => {
			link.onload = () => resolve();
			link.onerror = reject;
		});
	}

	static applyCSS(bundles: string[]): void {
		window.requestAnimationFrame(() => {
			const prefix = __riverClientGlobal.get("publicPathPrefix");

			for (const bundle of bundles) {
				// Check using the data attribute without escaping
				if (
					document.querySelector(
						`link[data-river-css-bundle="${bundle}"]`,
					)
				) {
					continue;
				}

				const link = document.createElement("link");
				link.rel = "stylesheet";
				link.href = prefix + bundle;
				link.setAttribute("data-river-css-bundle", bundle);
				document.head.appendChild(link);
			}
		});
	}
}
