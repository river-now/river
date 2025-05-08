import {
	addBuildIDListener,
	addLocationListener,
	addRouteChangeListener,
	applyScrollState,
	internal_RiverClientGlobal as ctx,
	getLocation,
	getRouterData,
	type RouteChangeEvent,
} from "river.now/client";
import {
	createEffect,
	createMemo,
	createRenderEffect,
	createSignal,
	type JSX,
	Show,
} from "solid-js";

/////////////////////////////////////////////////////////////////////
/////// CORE SETUP
/////////////////////////////////////////////////////////////////////

const [latestEvent, setLatestEvent] = createSignal<RouteChangeEvent | null>(null);
const [loadersData, setLoadersData] = createSignal(ctx.get("loadersData"));
const [clientLoadersData, setClientLoadersData] = createSignal(
	ctx.get("clientLoadersData"),
);
const [routerData, setRouterData] = createSignal(getRouterData());
const [outermostErrorIdx, setOutermostErrorIdx] = createSignal(
	ctx.get("outermostErrorIdx"),
);
const [outermostError, setOutermostError] = createSignal(ctx.get("outermostError"));

export { clientLoadersData, loadersData, routerData };

addRouteChangeListener((e) => {
	setLatestEvent(e);
	setLoadersData(ctx.get("loadersData"));
	setClientLoadersData(ctx.get("clientLoadersData"));
	setRouterData(getRouterData());
	setOutermostErrorIdx(ctx.get("outermostErrorIdx"));
	setOutermostError(ctx.get("outermostError"));
});

/////////////////////////////////////////////////////////////////////
/////// BUILD ID LISTENER
/////////////////////////////////////////////////////////////////////

addBuildIDListener((e) => {
	if (!e.detail.fromGETAction) {
		return;
	}
	setRouterData(getRouterData());
});

/////////////////////////////////////////////////////////////////////
/////// LOCATION
/////////////////////////////////////////////////////////////////////

const [location, setLocation] = createSignal(getLocation());

export { location };

addLocationListener(() => {
	setLocation(getLocation());
});

/////////////////////////////////////////////////////////////////////
/////// COMPONENT
/////////////////////////////////////////////////////////////////////

export function RiverRootOutlet(props: { idx?: number }): JSX.Element {
	const idx = props.idx ?? 0;

	if (idx === 0) {
		setClientLoadersData(ctx.get("clientLoadersData"));
	}

	const [currentImportURL, setCurrentImportURL] = createSignal(
		ctx.get("importURLs")?.[idx],
	);
	const [currentExportKey, setCurrentExportKey] = createSignal(
		ctx.get("exportKeys")?.[idx],
	);

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
		});
	}

	createRenderEffect(() => {
		const e = latestEvent();
		if (!e || idx !== 0) {
			return;
		}
		window.requestAnimationFrame(() => {
			applyScrollState(e.detail.scrollState);
		});
	});

	const isErrorIdxMemo = createMemo(() => {
		return idx === outermostErrorIdx();
	});

	const currentCompMemo = createMemo(() => {
		if (isErrorIdxMemo()) {
			return null;
		}
		currentImportURL();
		currentExportKey();
		return ctx.get("activeComponents")?.[idx];
	});

	const shouldFallbackOutletMemo = createMemo(() => {
		if (isErrorIdxMemo()) {
			return false;
		}
		if (currentCompMemo()) {
			return false;
		}
		return idx + 1 < loadersData().length;
	});

	const errorCompMemo = createMemo(() => {
		if (!isErrorIdxMemo()) {
			return null;
		}
		return ctx.get("activeErrorBoundary");
	});

	return (
		<>
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

			<Show when={isErrorIdxMemo()}>
				{errorCompMemo()?.({ error: outermostError() }) ??
					`Error: ${outermostError() || "unknown"}`}
			</Show>
		</>
	);
}
