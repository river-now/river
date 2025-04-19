import NProgress from "nprogress";
import { addStatusListener, getStatus, type StatusEvent } from "river.now/client";

let debounceStartTimer: number | null = null;
let debounceEndTimer: number | null = null;

addStatusListener((e) => {
	if (e.detail.isNavigating || e.detail.isRevalidating || e.detail.isSubmitting) {
		if (debounceStartTimer) {
			clearTimeout(debounceStartTimer);
		}
		debounceStartTimer = window.setTimeout(startNProgress, 30);
		return;
	}
	if (debounceEndTimer) {
		clearTimeout(debounceEndTimer);
	}
	debounceEndTimer = window.setTimeout(stopNProgress, 3);
});

function startNProgress() {
	if (!getIsWorking()) {
		return;
	}
	if (!NProgress.isStarted()) {
		NProgress.start();
	}
}

function stopNProgress() {
	if (getIsWorking()) {
		return;
	}
	if (NProgress.isStarted()) {
		NProgress.done();
	}
}

function getIsWorking(statusEvent?: StatusEvent) {
	const statusEventToUse = statusEvent?.detail || getStatus();
	return (
		statusEventToUse.isNavigating ||
		statusEventToUse.isRevalidating ||
		statusEventToUse.isSubmitting
	);
}
