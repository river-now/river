import { jsonDeepEquals } from "river.now/kit/json";
import { resolvePublicHref } from "./resolve_public_href.ts";
import { __riverClientGlobal } from "./river_ctx/river_ctx.ts";

export class ComponentLoader {
	static async loadComponents(
		importURLs: string[],
	): Promise<Map<string, any>> {
		const dedupedURLs = [...new Set(importURLs)];
		const modules = await Promise.all(
			dedupedURLs.map(async (url) => {
				if (!url) return undefined;
				return import(/* @vite-ignore */ resolvePublicHref(url));
			}),
		);

		return new Map(dedupedURLs.map((url, i) => [url, modules[i]]));
	}

	static async handleComponents(importURLs: string[]): Promise<void> {
		const modulesMap = await this.loadComponents(importURLs);
		const originalImportURLs = __riverClientGlobal.get("importURLs");
		const exportKeys = __riverClientGlobal.get("exportKeys") ?? [];

		// Build new components array
		const newActiveComponents = originalImportURLs.map(
			(url: string, i: number) => {
				const module = modulesMap.get(url);
				const key = exportKeys[i] ?? "default";
				return module?.[key] ?? null;
			},
		);

		// Only update if components actually changed
		if (
			!jsonDeepEquals(
				newActiveComponents,
				__riverClientGlobal.get("activeComponents"),
			)
		) {
			__riverClientGlobal.set("activeComponents", newActiveComponents);
		}

		// Handle error boundary
		const errorIdx = __riverClientGlobal.get("outermostErrorIdx");
		if (errorIdx != null) {
			const errorModuleURL = originalImportURLs[errorIdx];
			let errorComponent;

			if (errorModuleURL) {
				const errorModule = modulesMap.get(errorModuleURL);
				const errorKey = __riverClientGlobal.get("errorExportKey");
				if (errorKey && errorModule) {
					errorComponent = errorModule[errorKey];
				}
			}

			const newErrorBoundary =
				errorComponent ??
				__riverClientGlobal.get("defaultErrorBoundary");

			// Only update if changed
			const currentErrorBoundary = __riverClientGlobal.get(
				"activeErrorBoundary",
			);
			if (currentErrorBoundary !== newErrorBoundary) {
				__riverClientGlobal.set(
					"activeErrorBoundary",
					newErrorBoundary,
				);
			}
		}
	}
}
