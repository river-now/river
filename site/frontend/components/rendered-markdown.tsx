import { getHrefDetails } from "river.now/kit/url";
import { createEffect, onCleanup } from "solid-js";
import { render } from "solid-js/web";
import { waveRuntimeURL } from "../river.gen.ts";
import { Link } from "./app_link.tsx";
import { initHighlight } from "./highlight.ts";

const highlight = await initHighlight();

/**
 * Renders markdown with syntax highlighting and prefetching for internal links.
 * External links are opened in a new tab.
 * Uses a global map to keep track of prefetch handlers for cleanup, which means
 * that this component should only be used once per page.
 */
export function RenderedMarkdown(props: { markdown: string }) {
	let containerRef: HTMLDivElement | null = null;
	const disposers: Array<() => void> = [];

	// Cleanup function to remove any previously rendered components
	const cleanupPreviousRender = () => {
		disposers.forEach((dispose) => dispose());
		disposers.length = 0;
	};

	// Process the markdown content
	const processContent = () => {
		if (!containerRef) {
			return;
		}

		cleanupPreviousRender();

		containerRef.innerHTML = props.markdown; // Set the HTML content

		// Process code blocks
		const codeBlocks = containerRef.querySelectorAll("pre code");
		for (const codeBlock of codeBlocks) {
			highlight.highlightElement(codeBlock as HTMLElement);
		}

		// Process links
		for (const link of containerRef.querySelectorAll("a")) {
			const hrefDetails = getHrefDetails(link.href);

			if (hrefDetails.isHTTP && hrefDetails.isExternal) {
				link.dataset.external = "true";
				link.target = "_blank";
			} else {
				const href = link.href;
				const label = link.innerText;
				const placeholder = document.createElement("span");
				link.parentNode?.replaceChild(placeholder, link);

				const dispose = render(
					() => (
						<Link prefetch="intent" href={href}>
							{label}
						</Link>
					),
					placeholder,
				);
				disposers.push(dispose);
			}
		}

		// Process images
		for (const img of containerRef.querySelectorAll("img")) {
			// if data-src is set, grab value
			const src = img.getAttribute("data-src");
			if (src) {
				img.src = waveRuntimeURL(src as any);
				img.removeAttribute("data-src");
			}

			const width = img.getAttribute("data-width");
			const height = img.getAttribute("data-height");
			if (width && height) {
				img.style.aspectRatio = `${width}/${height}`;
			}
		}
	};

	// Set up ref callback to store the container element
	const ref = (el: HTMLDivElement | null) => {
		containerRef = el;
		if (el) {
			processContent();
		}
	};

	// Create effect to run processContent when markdown changes
	createEffect(() => {
		props.markdown; // Access props.markdown to track changes
		if (containerRef) {
			processContent();
		}
	});

	onCleanup(cleanupPreviousRender); // Clean up all disposers when component unmounts

	return <div ref={ref} class={"content"} />;
}
