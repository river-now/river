import { atom, createStore, Provider, useAtomValue } from "jotai";
import {
	type JSX,
	useEffect,
	useLayoutEffect,
	useMemo,
	useRef,
	useState,
} from "react";
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

/////////////////////////////////////////////////////////////////////
/////// JOTAI STORE
/////////////////////////////////////////////////////////////////////

const jotaiStore = createStore();

export function RiverProvider({ children }: React.PropsWithChildren): JSX.Element {
	return <Provider store={jotaiStore}>{children}</Provider>;
}

/////////////////////////////////////////////////////////////////////
/////// CORE SETUP
/////////////////////////////////////////////////////////////////////

const latestEventAtom = atom<RouteChangeEvent | null>(null);
const loadersDataAtom = atom(ctx.get("loadersData"));
const clientLoadersDataAtom = atom(ctx.get("clientLoadersData"));
const routerDataAtom = atom(getRouterData());
const outermostErrorIdxAtom = atom(ctx.get("outermostErrorIdx"));
const outermostErrorAtom = atom(ctx.get("outermostError"));

export { clientLoadersDataAtom, loadersDataAtom, routerDataAtom };

addRouteChangeListener((e) => {
	jotaiStore.set(latestEventAtom, e);
	jotaiStore.set(loadersDataAtom, ctx.get("loadersData"));
	jotaiStore.set(clientLoadersDataAtom, ctx.get("clientLoadersData"));
	jotaiStore.set(routerDataAtom, getRouterData());
	jotaiStore.set(outermostErrorIdxAtom, ctx.get("outermostErrorIdx"));
	jotaiStore.set(outermostErrorAtom, ctx.get("outermostError"));
});

/////////////////////////////////////////////////////////////////////
/////// BUILD ID LISTENER
/////////////////////////////////////////////////////////////////////

addBuildIDListener((e) => {
	if (!e.detail.fromGETAction) {
		return;
	}
	jotaiStore.set(routerDataAtom, getRouterData());
});

/////////////////////////////////////////////////////////////////////
/////// LOCATION
/////////////////////////////////////////////////////////////////////

const locationAtom = atom(getLocation());

export function useLocation() {
	return useAtomValue(locationAtom);
}

addLocationListener(() => {
	jotaiStore.set(locationAtom, getLocation());
});

/////////////////////////////////////////////////////////////////////
/////// COMPONENT
/////////////////////////////////////////////////////////////////////

export function RiverRootOutlet(props: { idx?: number }): JSX.Element {
	const idx = props.idx ?? 0;

	const initialRenderRef = useRef(true);
	if (idx === 0 && initialRenderRef.current) {
		initialRenderRef.current = false;
		jotaiStore.set(clientLoadersDataAtom, ctx.get("clientLoadersData"));
	}

	const latestEvent = useAtomValue(latestEventAtom);
	const outermostErrorIdx = useAtomValue(outermostErrorIdxAtom);
	const loadersData = useAtomValue(loadersDataAtom);
	const outermostError = useAtomValue(outermostErrorAtom);

	const [currentImportURL, setCurrentImportURL] = useState(
		ctx.get("importURLs")?.[idx],
	);
	const [currentExportKey, setCurrentExportKey] = useState(
		ctx.get("exportKeys")?.[idx],
	);
	const [nextImportURL, setNextImportURL] = useState(ctx.get("importURLs")?.[idx + 1]);
	const [nextExportKey, setNextExportKey] = useState(ctx.get("exportKeys")?.[idx + 1]);

	// biome-ignore lint/correctness/useExhaustiveDependencies: nope
	useEffect(() => {
		if (!currentImportURL || !latestEvent) {
			return;
		}

		const newCurrentImportURL = ctx.get("importURLs")?.[idx];
		const newCurrentExportKey = ctx.get("exportKeys")?.[idx];

		if (currentImportURL !== newCurrentImportURL) {
			setCurrentImportURL(newCurrentImportURL);
		}
		if (currentExportKey !== newCurrentExportKey) {
			setCurrentExportKey(newCurrentExportKey);
		}

		// these are also needed for Outlets to render correctly
		const newNextImportURL = ctx.get("importURLs")?.[idx + 1];
		const newNextExportKey = ctx.get("exportKeys")?.[idx + 1];

		if (nextImportURL !== newNextImportURL) {
			setNextImportURL(newNextImportURL);
		}
		if (nextExportKey !== newNextExportKey) {
			setNextExportKey(newNextExportKey);
		}
	}, [latestEvent]);

	useLayoutEffect(() => {
		if (!latestEvent || idx !== 0) {
			return;
		}
		window.requestAnimationFrame(() => {
			applyScrollState(latestEvent.detail.scrollState);
		});
	}, [latestEvent, idx]);

	const isErrorIdxMemo = useMemo(() => {
		return idx === outermostErrorIdx;
	}, [idx, outermostErrorIdx]);

	// biome-ignore lint/correctness/useExhaustiveDependencies: nope
	const CurrentCompMemo = useMemo(() => {
		if (isErrorIdxMemo) {
			return null;
		}
		return ctx.get("activeComponents")?.[idx];
	}, [isErrorIdxMemo, currentImportURL, currentExportKey]);

	// biome-ignore lint/correctness/useExhaustiveDependencies: nope
	const Outlet = useMemo(
		() => (localProps: Record<string, any> | undefined) => {
			return <RiverRootOutlet {...localProps} {...props} idx={idx + 1} />;
		},
		[nextImportURL, nextExportKey],
	);

	const shouldFallbackOutletMemo = useMemo(() => {
		if (isErrorIdxMemo) {
			return false;
		}
		if (CurrentCompMemo) {
			return false;
		}
		return idx + 1 < loadersData.length;
	}, [isErrorIdxMemo, CurrentCompMemo, idx, loadersData]);

	const ErrorCompMemo = useMemo(() => {
		if (!isErrorIdxMemo) {
			return null;
		}
		return ctx.get("activeErrorBoundary");
	}, [isErrorIdxMemo]);

	if (isErrorIdxMemo) {
		if (ErrorCompMemo) {
			return <ErrorCompMemo error={outermostError} />;
		}
		return <>{`Error: ${outermostError || "unknown"}`}</>;
	}

	if (!CurrentCompMemo) {
		if (shouldFallbackOutletMemo) {
			return <Outlet />;
		}
		return <></>;
	}

	return <CurrentCompMemo idx={idx} Outlet={Outlet} />;
}
