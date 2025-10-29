import {
	createPatternRegistry,
	registerPattern,
} from "river.now/kit/matcher/register";
import { setupClientLoaders } from "./client_loaders.ts";
import { ComponentLoader } from "./component_loader.ts";
import { defaultErrorBoundary } from "./error_boundary.ts";
import { RIVER_HARD_RELOAD_QUERY_PARAM } from "./hard_reload.ts";
import { HistoryManager } from "./history/history.ts";
import { initHMR } from "./hmr/hmr.ts";
import type { RiverAppConfig } from "./river_app_helpers/river_app_helpers.ts";
import {
	__riverClientGlobal,
	type RiverClientGlobal,
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
	const clientModuleMap: RiverClientGlobal["clientModuleMap"] = {};

	// Populate client module map with initial page's modules
	const initialMatchedPatterns =
		__riverClientGlobal.get("matchedPatterns") || [];
	const initialImportURLs = __riverClientGlobal.get("importURLs") || [];
	const initialExportKeys = __riverClientGlobal.get("exportKeys") || [];
	const initialErrorExportKeys =
		__riverClientGlobal.get("errorExportKeys") || [];

	for (let i = 0; i < initialMatchedPatterns.length; i++) {
		const pattern = initialMatchedPatterns[i];
		const importURL = initialImportURLs[i];
		const exportKey = initialExportKeys[i];
		const errorExportKey = initialErrorExportKeys[i];

		if (pattern && importURL) {
			clientModuleMap[pattern] = {
				importURL,
				exportKey: exportKey || "default",
				errorExportKey: errorExportKey || "",
			};
		}
	}
	__riverClientGlobal.set("clientModuleMap", clientModuleMap);

	const patternRegistry = createPatternRegistry({
		dynamicParamPrefixRune: options.riverAppConfig.loadersDynamicRune,
		splatSegmentRune: options.riverAppConfig.loadersSplatRune,
		explicitIndexSegment:
			options.riverAppConfig.loadersExplicitIndexSegment,
	});
	__riverClientGlobal.set("patternRegistry", patternRegistry);

	const manifestURL = __riverClientGlobal.get("routeManifestURL");
	if (manifestURL) {
		fetch(manifestURL)
			.then((response) => response.json())
			.then((manifest) => {
				__riverClientGlobal.set("routeManifest", manifest);

				// Register all patterns from manifest into the existing registry
				for (const pattern of Object.keys(manifest)) {
					registerPattern(patternRegistry, pattern);
				}
			})
			.catch((error) => {
				// This is no biggie -- it's a progressive enhancement
				console.warn("Failed to load route manifest:", error);
			});
	}

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

	const importURLs = __riverClientGlobal.get("importURLs");

	// Load initial components
	await ComponentLoader.handleComponents(importURLs);

	// Setup client loaders
	await setupClientLoaders();

	// Handle error boundary component (must come after setupClientLoaders)
	await ComponentLoader.handleErrorBoundaryComponent(importURLs);

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
