import { addOnWindowFocusListener } from "river.now/kit/listeners";
import {
	addStatusListener,
	getLastTriggeredNavOrRevalidateTimestampMS,
	getStatus,
	revalidate,
	type StatusEvent,
} from "./client.ts";

const DEFAULT_DELAY = 12;

type GlobalLoadingIndicatorIncludesOption =
	| "navigations"
	| "submissions"
	| "revalidations";

type GlobalLoadingIndicatorConfig = {
	start: () => void;
	stop: () => void;
	isRunning: () => boolean;
	include?: "all" | Array<GlobalLoadingIndicatorIncludesOption>;
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
	return isArray && config.include?.includes(includesOption);
}

export function setupGlobalLoadingIndicator(
	config: GlobalLoadingIndicatorConfig,
) {
	let gliDebounceStartTimer: number | null = null;
	let gliDebounceStopTimer: number | null = null;
	const includesAll = !config.include || config.include === "all";
	const pc: ParsedGlobalLoadingIndicatorConfig = {
		includesAll,
		includesNavigations:
			resolveIncludes(config, "navigations") || includesAll,
		includesSubmissions:
			resolveIncludes(config, "submissions") || includesAll,
		includesRevalidations:
			resolveIncludes(config, "revalidations") || includesAll,
		startDelayMS: config.startDelayMS ?? DEFAULT_DELAY,
		stopDelayMS: config.stopDelayMS ?? DEFAULT_DELAY,
	};
	function clearStartTimer() {
		if (gliDebounceStartTimer) {
			window.clearTimeout(gliDebounceStartTimer);
			gliDebounceStartTimer = null;
		}
	}
	function clearStopTimer() {
		if (gliDebounceStopTimer) {
			window.clearTimeout(gliDebounceStopTimer);
			gliDebounceStopTimer = null;
		}
	}
	function clearTimers() {
		clearStartTimer();
		clearStopTimer();
	}
	function handleStatusChange(e?: StatusEvent) {
		const shouldBeWorking = getIsWorking(pc, e);
		if (shouldBeWorking) {
			clearStopTimer();
			if (!gliDebounceStartTimer) {
				gliDebounceStartTimer = window.setTimeout(() => {
					gliDebounceStartTimer = null;
					if (!config.isRunning() && getIsWorking(pc)) {
						config.start();
					}
				}, pc.startDelayMS);
			}
		} else {
			clearStartTimer();
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
	handleStatusChange();
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
		return (
			status.isNavigating || status.isSubmitting || status.isRevalidating
		);
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

/**
 * If called, will setup listeners to revalidate the current route when
 * the window regains focus and at least `staleTimeMS` has passed since
 * the last revalidation. The `staleTimeMS` option defaults to 5,000
 * (5 seconds). Returns a cleanup function.
 */
export function revalidateOnWindowFocus(options?: { staleTimeMS?: number }) {
	const staleTimeMS = options?.staleTimeMS ?? 5_000;
	return addOnWindowFocusListener(() => {
		const status = getStatus();
		if (
			!status.isNavigating &&
			!status.isSubmitting &&
			!status.isRevalidating
		) {
			if (
				Date.now() - getLastTriggeredNavOrRevalidateTimestampMS() <
				staleTimeMS
			) {
				return;
			}
			revalidate();
		}
	});
}
