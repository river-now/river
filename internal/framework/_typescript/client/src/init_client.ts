import { setupClientLoaders } from "./client_loaders.ts";
import { ComponentLoader } from "./component_loader.ts";
import { defaultErrorBoundary } from "./error_boundary.ts";
import { RIVER_HARD_RELOAD_QUERY_PARAM } from "./hard_reload.ts";
import { HistoryManager } from "./history/history.ts";
import { initHMR } from "./hmr/hmr.ts";
import type { RiverAppConfig } from "./river_app_helpers/river_app_helpers.ts";
import {
	__riverClientGlobal,
	type RouteErrorComponent,
} from "./river_ctx/river_ctx.ts";
import { scrollStateManager } from "./scroll_state_manager.ts";

export async function initClient(options: {
	riverAppConfig: RiverAppConfig;
	renderFn: () => void;
	defaultErrorBoundary?: RouteErrorComponent;
	useViewTransitions?: boolean;
}): Promise<void> {
	initHMR();

	// Setup beforeunload handler for scroll restoration
	window.addEventListener("beforeunload", () => {
		scrollStateManager.savePageRefreshState();
	});

	__riverClientGlobal.set("riverAppConfig", options.riverAppConfig);

	// Set options
	if (options.defaultErrorBoundary) {
		__riverClientGlobal.set(
			"defaultErrorBoundary",
			options.defaultErrorBoundary,
		);
	} else {
		__riverClientGlobal.set("defaultErrorBoundary", defaultErrorBoundary);
	}

	if (options.useViewTransitions) {
		__riverClientGlobal.set("useViewTransitions", true);
	}

	// Initialize history
	HistoryManager.init();

	// Clean URL
	const url = new URL(window.location.href);
	if (url.searchParams.has(RIVER_HARD_RELOAD_QUERY_PARAM)) {
		url.searchParams.delete(RIVER_HARD_RELOAD_QUERY_PARAM);
		HistoryManager.getInstance().replace(url.href);
	}

	// Load initial components
	await ComponentLoader.handleComponents(
		__riverClientGlobal.get("importURLs"),
	);

	// Setup client loaders
	await setupClientLoaders();

	// Render
	options.renderFn();

	// Restore scroll
	scrollStateManager.restorePageRefreshState();

	// Touch detection
	window.addEventListener(
		"touchstart",
		() => {
			__riverClientGlobal.set("isTouchDevice", true);
		},
		{ once: true },
	);
}
