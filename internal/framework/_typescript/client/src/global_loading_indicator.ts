import { addStatusListener, getStatus, type StatusEvent } from "./client.ts";

type GlobalLoadingIndicatorIncludesOption =
	| "navigations"
	| "submissions"
	| "revalidations";

type GlobalLoadingIndicatorConfig = {
	include: "all" | Array<GlobalLoadingIndicatorIncludesOption>;
	start: () => void;
	stop: () => void;
	isRunning: () => boolean;
	startDelayMS?: number;
	stopDelayMS?: number;
};

type ParsedGlobalLoadingIndicatorConfig = {
	includesAll: boolean;
	includesNavigations: boolean;
	includesSubmissions: boolean;
	includesRevalidations: boolean;
	startDelayMS: number;
	stopDelayMS: number;
};

let gliDebounceStartTimer: number | null = null;
let gliDebounceStopTimer: number | null = null;

function resolveIncludes(
	config: GlobalLoadingIndicatorConfig,
	includesOption: GlobalLoadingIndicatorIncludesOption,
) {
	const isArray = Array.isArray(config.include);
	return isArray && config.include.includes(includesOption);
}

export function setupGlobalLoadingIndicator(config: GlobalLoadingIndicatorConfig) {
	const includesAll = config.include === "all";
	const pc: ParsedGlobalLoadingIndicatorConfig = {
		includesAll,
		includesNavigations: includesAll || resolveIncludes(config, "navigations"),
		includesSubmissions: includesAll || resolveIncludes(config, "submissions"),
		includesRevalidations: includesAll || resolveIncludes(config, "revalidations"),
		startDelayMS: config.startDelayMS ?? 30,
		stopDelayMS: config.stopDelayMS ?? 5,
	};
	function startLoader() {
		if (!getIsWorking(pc)) {
			return;
		}
		if (!config.isRunning()) {
			config.start();
		}
	}
	function stopLoader() {
		if (getIsWorking(pc)) {
			return;
		}
		if (config.isRunning()) {
			config.stop();
		}
	}
	addStatusListener((e) => {
		const isWorking = getIsWorking(pc, e);
		if (isWorking) {
			if (gliDebounceStartTimer) {
				clearTimeout(gliDebounceStartTimer);
			}
			gliDebounceStartTimer = window.setTimeout(startLoader, pc.startDelayMS);
		} else {
			if (gliDebounceStopTimer) {
				clearTimeout(gliDebounceStopTimer);
			}
			gliDebounceStopTimer = window.setTimeout(stopLoader, pc.stopDelayMS);
		}
	});
}

function getIsWorking(
	pc: ParsedGlobalLoadingIndicatorConfig,
	e?: StatusEvent,
): boolean {
	const status = e?.detail ?? getStatus();
	if (pc.includesAll) {
		return status.isNavigating || status.isSubmitting || status.isRevalidating;
	}
	if (pc.includesNavigations && status.isNavigating) {
		return true;
	}
	if (pc.includesSubmissions && status.isSubmitting) {
		return true;
	}
	if (pc.includesRevalidations && status.isRevalidating) {
		return true;
	}
	return false;
}
