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

const navigationStateAtom = atom({
	latestEvent: null as RouteChangeEvent | null,
	loadersData: ctx.get("loadersData"),
	clientLoadersData: ctx.get("clientLoadersData"),
	routerData: getRouterData(),
	outermostError: ctx.get("outermostError"),
	outermostErrorIdx: ctx.get("outermostErrorIdx"),
	activeComponents: ctx.get("activeComponents"),
	activeErrorBoundary: ctx.get("activeErrorBoundary"),
	importURLs: ctx.get("importURLs"),
	exportKeys: ctx.get("exportKeys"),
});

export const loadersDataAtom = atom((get) => {
	return get(navigationStateAtom).loadersData;
});
export const clientLoadersDataAtom = atom((get) => {
	return get(navigationStateAtom).clientLoadersData;
});
export const routerDataAtom = atom((get) => {
	return get(navigationStateAtom).routerData;
});

addRouteChangeListener((e) => {
	jotaiStore.set(navigationStateAtom, {
		latestEvent: e,
		loadersData: ctx.get("loadersData"),
		clientLoadersData: ctx.get("clientLoadersData"),
		routerData: getRouterData(),
		outermostError: ctx.get("outermostError"),
		outermostErrorIdx: ctx.get("outermostErrorIdx"),
		activeComponents: ctx.get("activeComponents"),
		activeErrorBoundary: ctx.get("activeErrorBoundary"),
		importURLs: ctx.get("importURLs"),
		exportKeys: ctx.get("exportKeys"),
	});
});

/////////////////////////////////////////////////////////////////////
/////// BUILD ID LISTENER
/////////////////////////////////////////////////////////////////////

addBuildIDListener((e) => {
	if (!e.detail.fromGETAction) {
		return;
	}
	jotaiStore.set(navigationStateAtom, (prev) => ({
		...prev,
		routerData: getRouterData(),
	}));
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
	const state = useAtomValue(navigationStateAtom);
	const {
		latestEvent,
		loadersData,
		outermostError,
		outermostErrorIdx,
		activeComponents,
		activeErrorBoundary,
		importURLs,
		exportKeys,
	} = state;

	if (idx === 0 && initialRenderRef.current) {
		initialRenderRef.current = false;
		jotaiStore.set(navigationStateAtom, (prev) => ({
			...prev,
			clientLoadersData: ctx.get("clientLoadersData"),
		}));
	}

	const [currentImportURL, setCurrentImportURL] = useState(importURLs?.[idx]);
	const [currentExportKey, setCurrentExportKey] = useState(exportKeys?.[idx]);
	const [nextImportURL, setNextImportURL] = useState(importURLs?.[idx + 1]);
	const [nextExportKey, setNextExportKey] = useState(exportKeys?.[idx + 1]);

	// biome-ignore lint/correctness/useExhaustiveDependencies: nope
	useEffect(() => {
		if (!currentImportURL || !latestEvent) {
			return;
		}

		const newCurrentImportURL = importURLs?.[idx];
		const newCurrentExportKey = exportKeys?.[idx];

		if (currentImportURL !== newCurrentImportURL) {
			setCurrentImportURL(newCurrentImportURL);
		}
		if (currentExportKey !== newCurrentExportKey) {
			setCurrentExportKey(newCurrentExportKey);
		}

		// these are also needed for Outlets to render correctly
		const newNextImportURL = importURLs?.[idx + 1];
		const newNextExportKey = exportKeys?.[idx + 1];

		if (nextImportURL !== newNextImportURL) {
			setNextImportURL(newNextImportURL);
		}
		if (nextExportKey !== newNextExportKey) {
			setNextExportKey(newNextExportKey);
		}
	}, [latestEvent, importURLs, exportKeys]);

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
		return activeComponents?.[idx];
	}, [isErrorIdxMemo, currentImportURL, currentExportKey, activeComponents]);

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
		return activeErrorBoundary;
	}, [isErrorIdxMemo, activeErrorBoundary]);

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
