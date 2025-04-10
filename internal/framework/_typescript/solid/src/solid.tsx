import {
	addRouteChangeListener,
	internal_RiverClientGlobal as ctx,
	type RiverRootOutletPropsGeneric,
	type RouteChangeEvent,
} from "@sjc5/river/client";
import { createEffect, createMemo, createSignal, ErrorBoundary, type JSX, Show } from "solid-js";

let shouldScroll = false;

const [latestEvent, setLatestEvent] = createSignal<RouteChangeEvent | null>(null);
const [loadersData, setLoadersData] = createSignal(ctx.get("loadersData"));
export { loadersData };

addRouteChangeListener((e) => {
	setLatestEvent(e);
	setLoadersData(ctx.get("loadersData"));
});

export function RiverRootOutlet(props: RiverRootOutletPropsGeneric<JSX.Element>): JSX.Element {
	const idx = props.idx ?? 0;
	const [currentImportURL, setCurrentImportURL] = createSignal(ctx.get("importURLs")?.[idx]);
	const [currentExportKey, setCurrentExportKey] = createSignal(ctx.get("exportKeys")?.[idx]);

	if (currentImportURL) {
		createEffect(() => {
			const e = latestEvent();
			if (!e) return;

			const newCurrentImportURL = ctx.get("importURLs")?.[idx];
			const newCurrentExportKey = ctx.get("exportKeys")?.[idx];

			if (currentImportURL() !== newCurrentImportURL) setCurrentImportURL(newCurrentImportURL);
			if (currentExportKey() !== newCurrentExportKey) setCurrentExportKey(newCurrentExportKey);

			if (idx === 0 && e.detail.scrollState) {
				shouldScroll = true;
				window.requestAnimationFrame(() => {
					if (shouldScroll && e.detail.scrollState) {
						window.scrollTo(e.detail.scrollState.x, e.detail.scrollState.y);
						shouldScroll = false;
					}
				});
			}
		});
	}

	const currentCompMemo = createMemo(() => {
		currentImportURL();
		currentExportKey();
		return ctx.get("activeComponents")?.[idx];
	});

	return (
		<ErrorBoundary fallback={<div>ERROR</div>}>
			<Show when={currentCompMemo()}>
				{currentCompMemo()({
					idx: idx,
					Outlet: (localProps: Record<string, any> | undefined) => {
						return <RiverRootOutlet {...localProps} {...props} idx={idx + 1} />;
					},
				})}
			</Show>
		</ErrorBoundary>
	);
}
