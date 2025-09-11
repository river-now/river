import { debounce } from "river.now/kit/debounce";
import { revalidate } from "../client.ts";
import { setupClientLoaders } from "../client_loaders.ts";
import { dispatchRouteChangeEvent } from "../events.ts";
import { __riverClientGlobal } from "../river_ctx/river_ctx.ts";
import { logInfo } from "../utils/logging.ts";

let devTimeSetupClientLoadersDebounced: () => Promise<void> = () =>
	Promise.resolve();

let hmrRevalidateSet: Set<string>;

export let __runClientLoadersAfterHMRUpdate: (
	importMeta: ImportMeta,
	pattern: string,
) => void = () => {};

export function initHMR() {
	if (import.meta.env.DEV) {
		(window as any).__waveRevalidate = revalidate;

		devTimeSetupClientLoadersDebounced = debounce(async () => {
			await setupClientLoaders();
			dispatchRouteChangeEvent({});
		}, 10);

		__runClientLoadersAfterHMRUpdate = (importMeta, pattern) => {
			if (hmrRevalidateSet === undefined) {
				hmrRevalidateSet = new Set();
			}

			if (import.meta.env.DEV && import.meta.hot) {
				const thisURL = new URL(importMeta.url, location.href);
				thisURL.search = "";
				const thisPathname = thisURL.pathname;

				const alreadyRegistered = hmrRevalidateSet.has(thisPathname);
				if (alreadyRegistered) {
					return;
				}

				hmrRevalidateSet.add(thisPathname);

				import.meta.hot.on("vite:afterUpdate", (props) => {
					for (const update of props.updates) {
						if (update.type === "js-update") {
							const updateURL = new URL(
								update.path,
								location.href,
							);
							updateURL.search = "";
							if (updateURL.pathname === thisURL.pathname) {
								if (
									__riverClientGlobal
										.get("matchedPatterns")
										.includes(pattern)
								) {
									logInfo(
										"Refreshing client loaders due to change in pattern:",
										pattern,
									);
									devTimeSetupClientLoadersDebounced();
								}
							}
						}
					}
				});
			}
		};
	}
}
