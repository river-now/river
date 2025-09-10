import { debounce } from "river.now/kit/debounce";
import {
	RIVER_ROUTE_CHANGE_EVENT_KEY,
	revalidate,
	setupClientLoaders,
} from "./client.ts";

let devTimeSetupClientLoadersDebounced: () => Promise<void> = () =>
	Promise.resolve();
let hmrRevalidateSet: Set<string>;
export let hmrRunClientLoaders: (importMeta: ImportMeta) => void = () => {};

export function initHMR() {
	if (import.meta.env.DEV) {
		(window as any).__waveRevalidate = revalidate;
		devTimeSetupClientLoadersDebounced = debounce(async () => {
			await setupClientLoaders();
			window.dispatchEvent(
				new CustomEvent(RIVER_ROUTE_CHANGE_EVENT_KEY, { detail: {} }),
			);
		}, 10);

		hmrRunClientLoaders = (importMeta: ImportMeta) => {
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
								devTimeSetupClientLoadersDebounced();
							}
						}
					}
				});
			}
		};
	}
}
