import { getHrefDetails, getIsGETRequest } from "river.now/kit/url";
import { LogInfo } from "./utils.ts";

export type RedirectData = { href: string } & (
	| {
			status: "did";
	  }
	| {
			status: "should";
			shouldRedirectStrategy: "hard" | "soft";
	  }
);

export function parseFetchResponseForRedirectData(
	reqInit: RequestInit,
	res: Response,
): RedirectData | null {
	if (res.redirected) {
		const newURL = new URL(res.url, window.location.href);
		const hrefDetails = getHrefDetails(newURL.href);
		if (!hrefDetails.isHTTP) {
			return null;
		}

		const isCurrent = newURL.href === window.location.href;
		if (isCurrent) {
			return { status: "did", href: newURL.href };
		}

		const wasGETRequest = getIsGETRequest(reqInit);
		if (!wasGETRequest) {
			LogInfo("Not a GET request. No way to handle.");
			return null;
		}

		return {
			status: "should",
			href: newURL.href,
			shouldRedirectStrategy: hrefDetails.isInternal ? "soft" : "hard",
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
		status: "should",
		href: hrefDetails.absoluteURL,
		shouldRedirectStrategy: hrefDetails.isInternal ? "soft" : "hard",
	};
}
