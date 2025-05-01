import {
	addBuildIDListener,
	addRouteChangeListener,
	internal_RiverClientGlobal as ctx,
	getCurrentRiverData,
	type RiverRootOutletPropsGeneric,
	type RouteChangeEvent,
} from "river.now/client";
import {
	createEffect,
	createMemo,
	createSignal,
	ErrorBoundary,
	type JSX,
	Show,
} from "solid-js";

let shouldScroll = false;

const [latestEvent, setLatestEvent] = createSignal<RouteChangeEvent | null>(null);
const [loadersData, setLoadersData] = createSignal(ctx.get("loadersData"));
const [clientLoadersData, setClientLoadersData] = createSignal(
	ctx.get("clientLoadersData"),
);
export { clientLoadersData, loadersData };

const [currentRiverData, setCurrentRiverData] = createSignal(getCurrentRiverData());
export { currentRiverData };
addRouteChangeListener(() => setCurrentRiverData(getCurrentRiverData()));

const [outermostErrorIdx, setOutermostErrorIdx] = createSignal(
	ctx.get("outermostErrorIdx"),
);
const [outermostError, setOutermostError] = createSignal(ctx.get("outermostError"));

addRouteChangeListener((e) => {
	setLatestEvent(e);
	setLoadersData(ctx.get("loadersData"));
	setClientLoadersData(ctx.get("clientLoadersData"));
	setCurrentRiverData(getCurrentRiverData());
	setOutermostErrorIdx(ctx.get("outermostErrorIdx"));
	setOutermostError(ctx.get("outermostError"));
});

addBuildIDListener((e) => {
	if (!e.detail.fromGETAction) {
		return;
	}
	setCurrentRiverData(getCurrentRiverData());
});

export function RiverRootOutlet(
	props: RiverRootOutletPropsGeneric<JSX.Element>,
): JSX.Element {
	const idx = props.idx ?? 0;

	const [currentImportURL, setCurrentImportURL] = createSignal(
		ctx.get("importURLs")?.[idx],
	);
	const [currentExportKey, setCurrentExportKey] = createSignal(
		ctx.get("exportKeys")?.[idx],
	);

	if (idx === 0) {
		setClientLoadersData(ctx.get("clientLoadersData"));
	}

	if (currentImportURL) {
		createEffect(() => {
			const e = latestEvent();
			if (!e) {
				return;
			}

			const newCurrentImportURL = ctx.get("importURLs")?.[idx];
			const newCurrentExportKey = ctx.get("exportKeys")?.[idx];

			if (currentImportURL() !== newCurrentImportURL) {
				setCurrentImportURL(newCurrentImportURL);
			}
			if (currentExportKey() !== newCurrentExportKey) {
				setCurrentExportKey(newCurrentExportKey);
			}

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

	const isErrorIdx = createMemo(() => {
		return idx === outermostErrorIdx();
	});

	const currentCompMemo = createMemo(() => {
		if (isErrorIdx()) {
			return null;
		}
		currentImportURL();
		currentExportKey();
		return ctx.get("activeComponents")?.[idx];
	});

	const shouldFallbackOutletMemo = createMemo(() => {
		if (isErrorIdx()) {
			return false;
		}
		if (currentCompMemo()) {
			return false;
		}
		return idx + 1 < loadersData().length;
	});

	const errorCompMemo = createMemo(() => {
		if (!isErrorIdx()) {
			return null;
		}
		return ctx.get("activeErrorBoundary");
	});

	return (
		<ErrorBoundary fallback={"Client error."}>
			<Show when={currentCompMemo()}>
				{currentCompMemo()({
					idx: idx,
					Outlet: (localProps: Record<string, any> | undefined) => {
						return <RiverRootOutlet {...localProps} {...props} idx={idx + 1} />;
					},
				})}
			</Show>

			<Show when={shouldFallbackOutletMemo()}>
				<RiverRootOutlet {...props} idx={idx + 1} />
			</Show>

			<Show when={isErrorIdx()}>{errorCompMemo()?.({ error: outermostError() })}</Show>
		</ErrorBoundary>
	);
}
