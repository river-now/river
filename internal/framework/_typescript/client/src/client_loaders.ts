import { findNestedMatches } from "river.now/kit/matcher/find-nested";
import { registerPattern } from "river.now/kit/matcher/register";
import { ComponentLoader } from "./component_loader.ts";
import {
	__riverClientGlobal,
	type GetRouteDataOutput,
} from "./river_ctx/river_ctx.ts";
import { isAbortError } from "./utils/errors.ts";
import { logError } from "./utils/logging.ts";

export async function setupClientLoaders(): Promise<void> {
	const clientLoadersData = await runWaitFns(
		{
			hasRootData: __riverClientGlobal.get("hasRootData"),
			importURLs: __riverClientGlobal.get("importURLs"),
			loadersData: __riverClientGlobal.get("loadersData"),
			matchedPatterns: __riverClientGlobal.get("matchedPatterns"),
			params: __riverClientGlobal.get("params"),
			splatValues: __riverClientGlobal.get("splatValues"),
		},
		__riverClientGlobal.get("buildID"),
		new AbortController().signal,
	);

	__riverClientGlobal.set("clientLoadersData", clientLoadersData);
}

export async function __registerClientLoaderPattern(
	pattern: string,
): Promise<void> {
	registerPattern(__riverClientGlobal.get("patternRegistry"), pattern);
}

// This is needed because the matcher, by definition, will only
// match when you have a full path match. If the path you are
// testing is longer than the registered patterns, you will get
// no match, even if some registered patterns would potentially
// be in the parent segments. This fixes that.
export async function findPartialMatchesOnClient(pathname: string) {
	const patternToWaitFnMap = __riverClientGlobal.get("patternToWaitFnMap");
	if (Object.keys(patternToWaitFnMap).length === 0) {
		return null;
	}

	const patternRegistry = __riverClientGlobal.get("patternRegistry");

	// First try the full path
	const fullResult = findNestedMatches(patternRegistry, pathname);
	if (fullResult) {
		// If we get a full match, we have everything we need
		return fullResult;
	}

	// If no full match, try progressively shorter paths to find partial matches
	const segments = pathname.split("/").filter(Boolean);

	// Try from longest to shortest
	for (let i = segments.length; i >= 0; i--) {
		const partialPath =
			i === 0 ? "/" : "/" + segments.slice(0, i).join("/");
		const result = findNestedMatches(patternRegistry, partialPath);
		if (result) {
			return result; // First match is the longest
		}
	}

	return null;
}

type PartialWaitFnJSON = Pick<
	GetRouteDataOutput,
	| "matchedPatterns"
	| "splatValues"
	| "params"
	| "hasRootData"
	| "loadersData"
	| "importURLs"
>;

async function runWaitFns(
	json: PartialWaitFnJSON,
	buildID: string,
	signal: AbortSignal,
): Promise<Array<any>> {
	await ComponentLoader.loadComponents(json.importURLs);

	const matchedPatterns = json.matchedPatterns ?? [];
	const patternToWaitFnMap = __riverClientGlobal.get("patternToWaitFnMap");
	const waitFnPromises: Array<Promise<any>> = [];

	let i = 0;
	for (const pattern of matchedPatterns) {
		if (patternToWaitFnMap[pattern]) {
			const serverDataPromise = Promise.resolve({
				matchedPatterns: json.matchedPatterns,
				loaderData: json.loadersData[i],
				rootData: json.hasRootData ? json.loadersData[0] : null,
				buildID: buildID,
			});

			waitFnPromises.push(
				patternToWaitFnMap[pattern]({
					params: json.params || {},
					splatValues: json.splatValues || [],
					serverDataPromise,
					signal,
				}),
			);
		} else {
			waitFnPromises.push(Promise.resolve());
		}
		i++;
	}

	return Promise.all(waitFnPromises);
}

export async function completeClientLoaders(
	json: PartialWaitFnJSON,
	buildID: string,
	runningLoaders: Map<string, Promise<any>>,
	signal: AbortSignal,
): Promise<Array<any>> {
	await ComponentLoader.loadComponents(json.importURLs);

	const matchedPatterns = json.matchedPatterns ?? [];
	const patternToWaitFnMap = __riverClientGlobal.get("patternToWaitFnMap");
	const finalPromises: Array<Promise<any>> = [];

	let i = 0;
	for (const pattern of matchedPatterns) {
		if (runningLoaders.has(pattern)) {
			finalPromises.push(runningLoaders.get(pattern)!);
		} else if (patternToWaitFnMap[pattern]) {
			const serverDataPromise = Promise.resolve({
				matchedPatterns: json.matchedPatterns,
				loaderData: json.loadersData[i],
				rootData: json.hasRootData ? json.loadersData[0] : null,
				buildID: buildID,
			});

			const loaderPromise = patternToWaitFnMap[pattern]({
				splatValues: json.splatValues || [],
				params: json.params || {},
				serverDataPromise,
				signal,
			}).catch((error: any) => {
				if (!isAbortError(error)) {
					logError(
						`Client loader error for pattern ${pattern}:`,
						error,
					);
				}
				return undefined;
			});
			finalPromises.push(loaderPromise);
		} else {
			finalPromises.push(Promise.resolve());
		}
		i++;
	}

	return Promise.all(finalPromises);
}
