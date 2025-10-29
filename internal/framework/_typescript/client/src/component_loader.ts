import { jsonDeepEquals } from "river.now/kit/json";
import { resolvePublicHref } from "./resolve_public_href.ts";
import { __riverClientGlobal } from "./river_ctx/river_ctx.ts";

export function getEffectiveErrorData(): {
	index: number | undefined;
	error: string | undefined;
} {
	const serverErrorIdx = __riverClientGlobal.get("outermostServerErrorIdx");
	const clientErrorIdx = __riverClientGlobal.get("outermostClientErrorIdx");
	let errorIdx: number | undefined;
	if (serverErrorIdx != null && clientErrorIdx != null) {
		errorIdx = Math.min(serverErrorIdx, clientErrorIdx);
	} else {
		errorIdx = serverErrorIdx ?? clientErrorIdx;
	}
	return {
		index: errorIdx,
		error:
			errorIdx === serverErrorIdx
				? __riverClientGlobal.get("outermostServerError")
				: errorIdx === clientErrorIdx
					? __riverClientGlobal.get("outermostClientError")
					: undefined,
	};
}

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
	}

	static async handleErrorBoundaryComponent(
		importURLs: string[],
	): Promise<void> {
		const modulesMap = await this.loadComponents(importURLs);
		const originalImportURLs = __riverClientGlobal.get("importURLs");

		// Handle error boundary
		const errorIdx = getEffectiveErrorData().index;

		if (errorIdx != null) {
			const errorModuleURL = originalImportURLs[errorIdx];
			let errorComponent;

			if (errorModuleURL) {
				const errorModule = modulesMap.get(errorModuleURL);
				const errorKeys = __riverClientGlobal.get("errorExportKeys");
				const errorKey = errorKeys ? errorKeys[errorIdx] : null;
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
