import { addStatusListener, getStatus, type StatusEvent } from "./client.ts";

const DEFAULT_START_DELAY_MS = 20;
const DEFAULT_STOP_DELAY_MS = 5;

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

function resolveIncludes(
	config: GlobalLoadingIndicatorConfig,
	includesOption: GlobalLoadingIndicatorIncludesOption,
) {
	const isArray = Array.isArray(config.include);
	return isArray && config.include.includes(includesOption);
}

export function setupGlobalLoadingIndicator(config: GlobalLoadingIndicatorConfig) {
	let gliDebounceStartTimer: number | null = null;
	let gliDebounceStopTimer: number | null = null;
	const includesAll = config.include === "all";
	const pc: ParsedGlobalLoadingIndicatorConfig = {
		includesAll,
		includesNavigations: includesAll || resolveIncludes(config, "navigations"),
		includesSubmissions: includesAll || resolveIncludes(config, "submissions"),
		includesRevalidations: includesAll || resolveIncludes(config, "revalidations"),
		startDelayMS: config.startDelayMS ?? DEFAULT_START_DELAY_MS,
		stopDelayMS: config.stopDelayMS ?? DEFAULT_STOP_DELAY_MS,
	};
	function clearTimers() {
		if (gliDebounceStartTimer) {
			window.clearTimeout(gliDebounceStartTimer);
			gliDebounceStartTimer = null;
		}
		if (gliDebounceStopTimer) {
			window.clearTimeout(gliDebounceStopTimer);
			gliDebounceStopTimer = null;
		}
	}
	function handleStatusChange(e?: StatusEvent) {
		const shouldBeWorking = getIsWorking(pc, e);
		if (shouldBeWorking) {
			// Clear stop timer if transitioning to working
			if (gliDebounceStopTimer) {
				window.clearTimeout(gliDebounceStopTimer);
				gliDebounceStopTimer = null;
			}
			// Only set start timer if not already set
			if (!gliDebounceStartTimer) {
				gliDebounceStartTimer = window.setTimeout(() => {
					gliDebounceStartTimer = null;
					if (!config.isRunning() && getIsWorking(pc)) {
						config.start();
					}
				}, pc.startDelayMS);
			}
		} else {
			// Clear start timer if transitioning to not working
			if (gliDebounceStartTimer) {
				window.clearTimeout(gliDebounceStartTimer);
				gliDebounceStartTimer = null;
			}
			// Only set stop timer if not already set
			if (!gliDebounceStopTimer) {
				gliDebounceStopTimer = window.setTimeout(() => {
					gliDebounceStopTimer = null;
					if (config.isRunning() && !getIsWorking(pc)) {
						config.stop();
					}
				}, pc.stopDelayMS);
			}
		}
	}
	// Check initial state
	handleStatusChange();
	// Listen for changes
	const removeStatusListenerCallback = addStatusListener(handleStatusChange);
	return () => {
		removeStatusListenerCallback();
		clearTimers();
		if (config.isRunning()) {
			config.stop();
		}
	};
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
