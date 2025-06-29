import {
	getHrefDetails,
	getIsGETRequest,
	type HrefDetails,
} from "river.now/kit/url";
import { LogInfo } from "./utils.ts";

export type RedirectData = { href: string; hrefDetails: HrefDetails } & (
	| {
			status: "did";
	  }
	| {
			status: "should";
			shouldRedirectStrategy: "hard" | "soft";
			latestBuildID: string;
	  }
);

export function getBuildIDFromResponse(response: Response | undefined) {
	return response?.headers.get("X-River-Build-Id") || "";
}

export function parseFetchResponseForRedirectData(
	reqInit: RequestInit,
	res: Response,
): RedirectData | null {
	const latestBuildID = getBuildIDFromResponse(res);

	const riverReloadTarget = res.headers.get("X-River-Reload");
	if (riverReloadTarget) {
		const newURL = new URL(riverReloadTarget, window.location.href);
		const hrefDetails = getHrefDetails(newURL.href);
		if (!hrefDetails.isHTTP) {
			return null;
		}

		return {
			hrefDetails,
			status: "should",
			href: riverReloadTarget,
			shouldRedirectStrategy: "hard",
			latestBuildID,
		};
	}

	if (res.redirected) {
		const newURL = new URL(res.url, window.location.href);
		const hrefDetails = getHrefDetails(newURL.href);
		if (!hrefDetails.isHTTP) {
			return null;
		}

		const isCurrent = newURL.href === window.location.href;
		if (isCurrent) {
			return { hrefDetails, status: "did", href: newURL.href };
		}

		const wasGETRequest = getIsGETRequest(reqInit);
		if (!wasGETRequest) {
			LogInfo("Not a GET request. No way to handle.");
			return null;
		}

		return {
			hrefDetails,
			status: "should",
			href: newURL.href,
			shouldRedirectStrategy: hrefDetails.isInternal ? "soft" : "hard",
			latestBuildID,
		};
	}

	const clientRedirectHeader = res.headers.get("X-Client-Redirect");

	if (!clientRedirectHeader) {
		return null;
	}

	const newURL = new URL(clientRedirectHeader, window.location.href);
	const hrefDetails = getHrefDetails(newURL.href);
	if (!hrefDetails.isHTTP) {
		return null;
	}

	return {
		hrefDetails,
		status: "should",
		href: hrefDetails.absoluteURL,
		shouldRedirectStrategy: hrefDetails.isInternal ? "soft" : "hard",
		latestBuildID,
	};
}
