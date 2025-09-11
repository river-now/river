import { getAnchorDetailsFromEvent, getHrefDetails } from "river.now/kit/url";
import { navigationStateManager, riverNavigate } from "./client.ts";
import { saveScrollState } from "./scroll_state_manager.ts";

type LinkOnClickCallback<E extends Event> = (event: E) => void | Promise<void>;

type LinkOnClickCallbacks<E extends Event> = {
	beforeBegin?: LinkOnClickCallback<E>;
	beforeRender?: LinkOnClickCallback<E>;
	afterRender?: LinkOnClickCallback<E>;
};

type GetPrefetchHandlersInput<E extends Event> = LinkOnClickCallbacks<E> & {
	href: string;
	delayMs?: number;
	scrollToTop?: boolean;
	replace?: boolean;
	search?: string;
	hash?: string;
	state?: unknown;
};

export function __getPrefetchHandlers<E extends Event>(
	input: GetPrefetchHandlersInput<E>,
) {
	const hrefDetails = getHrefDetails(input.href);
	if (!hrefDetails.isHTTP) {
		return;
	}

	// TypeScript type guard -- after this check, we know relativeURL exists
	const { relativeURL } = hrefDetails;
	if (!relativeURL || hrefDetails.isExternal) {
		return;
	}

	let timer: number | undefined;
	let prefetchStarted = false;
	const delayMs = input.delayMs ?? 100;

	async function prefetch(e: E): Promise<void> {
		if (prefetchStarted) return;
		prefetchStarted = true;

		if (input.beforeBegin) {
			await input.beforeBegin(e);
		}

		const fullUrl = new URL(relativeURL, window.location.href);
		if (input.search !== undefined) fullUrl.search = input.search;
		if (input.hash !== undefined) fullUrl.hash = input.hash;

		// Use the navigation system
		await navigationStateManager.navigate({
			href: fullUrl.href,
			navigationType: "prefetch",
			state: input.state,
		});
	}

	function start(e: E): void {
		if (prefetchStarted) return;
		timer = window.setTimeout(() => prefetch(e), delayMs);
	}

	function stop(): void {
		if (timer) {
			clearTimeout(timer);
			timer = undefined;
		}

		// Abort prefetch if it exists and hasn't been upgraded
		const targetUrl = new URL(relativeURL, window.location.href).href;
		const nav = navigationStateManager.getNavigation(targetUrl);
		if (nav && nav.type === "prefetch" && nav.intent === "none") {
			nav.control.abortController?.abort();
			navigationStateManager.removeNavigation(targetUrl);
		}

		prefetchStarted = false;
	}

	async function onClick(e: E): Promise<void> {
		if (e.defaultPrevented) return;

		const anchorDetails = getAnchorDetailsFromEvent(
			e as unknown as MouseEvent,
		);
		if (!anchorDetails) return;

		const { isEligibleForDefaultPrevention, isInternal } = anchorDetails;
		if (!isEligibleForDefaultPrevention || !isInternal) return;

		if (isJustAHashChange(anchorDetails)) {
			saveScrollState();
			return;
		}

		e.preventDefault();

		if (timer) {
			clearTimeout(timer);
			timer = undefined;
		}

		// Execute callbacks
		if (input.beforeBegin && !prefetchStarted) {
			await input.beforeBegin(e);
		}

		if (input.beforeRender) {
			await input.beforeRender(e);
		}

		// Use standard navigation -- it will upgrade the prefetch if it exists
		await riverNavigate(relativeURL, {
			scrollToTop: input.scrollToTop,
			replace: input.replace,
			search: input.search,
			hash: input.hash,
			state: input.state,
		});

		if (input.afterRender) {
			await input.afterRender(e);
		}
	}

	return {
		...hrefDetails,
		start,
		stop,
		onClick,
	};
}

export function __makeLinkOnClickFn<E extends Event>(
	callbacks: LinkOnClickCallbacks<E> & {
		scrollToTop?: boolean;
		replace?: boolean;
		state?: unknown;
	},
) {
	return async (e: E) => {
		if (e.defaultPrevented) return;

		const anchorDetails = getAnchorDetailsFromEvent(
			e as unknown as MouseEvent,
		);
		if (!anchorDetails) return;

		const { anchor, isEligibleForDefaultPrevention, isInternal } =
			anchorDetails;
		if (!anchor) return;

		if (isJustAHashChange(anchorDetails)) {
			saveScrollState();
			return;
		}

		if (isEligibleForDefaultPrevention && isInternal) {
			e.preventDefault();

			await callbacks.beforeBegin?.(e);

			const control = navigationStateManager.beginNavigation({
				href: anchor.href,
				navigationType: "userNavigation",
				scrollToTop: callbacks.scrollToTop,
				replace: callbacks.replace,
				state: callbacks.state,
			});

			if (!control.promise) return;

			const res = await control.promise;

			if (!res) {
				// If not here, loading indicator can get stuck on
				// following redirects
				const targetUrl = new URL(anchor.href, window.location.href)
					.href;
				navigationStateManager.removeNavigation(targetUrl);
				return;
			}

			await callbacks.beforeRender?.(e);

			const targetUrl = new URL(anchor.href, window.location.href).href;
			const entry = navigationStateManager.getNavigation(targetUrl);
			if (entry) {
				await navigationStateManager["processNavigationResult"](
					res,
					entry,
				);
			}

			await callbacks.afterRender?.(e);
		}
	};
}

function isJustAHashChange(
	anchorDetails: ReturnType<typeof getAnchorDetailsFromEvent>,
): boolean {
	if (!anchorDetails) return false;

	const { pathname, search, hash } = new URL(
		anchorDetails.anchor.href,
		window.location.href,
	);

	return !!(
		hash &&
		pathname === window.location.pathname &&
		search === window.location.search
	);
}
