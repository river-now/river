import { atom, useAtom, useAtomValue, useSetAtom } from "jotai";
import { type JSX, useEffect, useLayoutEffect, useMemo, useState } from "react";
import { flushSync } from "react-dom";
import {
	addBuildIDListener,
	addLocationListener,
	addRouteChangeListener,
	applyScrollState,
	internal_RiverClientGlobal as ctx,
	getCurrentRiverData,
	getLocation,
	type RiverRootOutletPropsGeneric,
	type RouteChangeEvent,
} from "river.now/client";
import { jsonDeepEquals } from "river.now/kit/json";

const importURLsAtom = atom(ctx.get("importURLs"));
const rootDataAtom = atom(ctx.get("hasRootData") ? ctx.get("loadersData")[0] : null);
const paramsAtom = atom(ctx.get("params") ?? {});
const splatValuesAtom = atom(ctx.get("splatValues") ?? []);
export const loadersDataAtom = atom(ctx.get("loadersData"));
export const clientLoadersDataAtom = atom(ctx.get("clientLoadersData"));
export const currentRiverDataAtom = atom(getCurrentRiverData());

const outermostErrorIdxAtom = atom(ctx.get("outermostErrorIdx"));
const outermostErrorAtom = atom(ctx.get("outermostError"));

const latestEventAtom = atom<RouteChangeEvent | null>(null);

const locationAtom = atom(getLocation());

export function useLocation() {
	return useAtomValue(locationAtom);
}

export function RiverRootOutlet(
	props: RiverRootOutletPropsGeneric<JSX.Element>,
): JSX.Element {
	const idx = props.idx ?? 0;
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
		return addRouteChangeListener(() => {
			const newCurrentImportURL = ctx.get("importURLs")?.[idx];
			const newCurrentExportKey = ctx.get("exportKeys")?.[idx];
			const newNextImportURL = ctx.get("importURLs")?.[idx + 1];
			const newNextExportKey = ctx.get("exportKeys")?.[idx + 1];

			flushSync(() => {
				if (currentImportURL !== newCurrentImportURL) {
					setCurrentImportURL(newCurrentImportURL);
				}
				if (currentExportKey !== newCurrentExportKey) {
					setCurrentExportKey(newCurrentExportKey);
				}
				if (nextImportURL !== newNextImportURL) {
					setNextImportURL(newNextImportURL);
				}
				if (nextExportKey !== newNextExportKey) {
					setNextExportKey(newNextExportKey);
				}
			});
		});
	}, [currentImportURL, currentExportKey]);

	const [importURLs, setImportURLs] = useAtom(importURLsAtom);
	const [rootData, setRootData] = useAtom(rootDataAtom);
	const [params, setParams] = useAtom(paramsAtom);
	const [splatValues, setSplatValues] = useAtom(splatValuesAtom);
	const [loadersData, setLoadersData] = useAtom(loadersDataAtom);
	const [clientLoadersData, setClientLoadersData] = useAtom(clientLoadersDataAtom);
	const [currentRiverData, setCurrentRiverData] = useAtom(currentRiverDataAtom);

	const [outermostErrorIdx, setOutermostErrorIdx] = useAtom(outermostErrorIdxAtom);
	const [outermostError, setOutermostError] = useAtom(outermostErrorAtom);

	const [latestEvent, setLatestEvent] = useAtom(latestEventAtom);

	useEffect(() => {
		if (idx === 0) {
			setClientLoadersData(ctx.get("clientLoadersData"));
		}
	}, []);

	// biome-ignore lint/correctness/useExhaustiveDependencies: nope
	useEffect(() => {
		if (idx === 0) {
			return addRouteChangeListener((e) => {
				const newImportURLs = ctx.get("importURLs");
				const newRootData = ctx.get(
					ctx.get("hasRootData") ? ctx.get("loadersData")[0] : null,
				);
				const newParams = ctx.get("params") ?? {};
				const newSplatValues = ctx.get("splatValues") ?? [];
				const newLoadersData = ctx.get("loadersData");
				const newClientLoadersData = ctx.get("clientLoadersData");
				const newCurrentRiverData = getCurrentRiverData();

				const newOutermostErrorIdx = ctx.get("outermostErrorIdx");
				const newOutermostError = ctx.get("outermostError");

				flushSync(() => {
					if (!jsonDeepEquals(importURLs, newImportURLs)) {
						setImportURLs(newImportURLs);
					}
					if (!jsonDeepEquals(rootData, newRootData)) {
						setRootData(newRootData);
					}
					if (!jsonDeepEquals(params, newParams)) {
						setParams(newParams);
					}
					if (!jsonDeepEquals(splatValues, newSplatValues)) {
						setSplatValues(newSplatValues);
					}
					if (!jsonDeepEquals(loadersData, newLoadersData)) {
						setLoadersData(newLoadersData);
					}
					if (!jsonDeepEquals(clientLoadersData, newClientLoadersData)) {
						setClientLoadersData(newClientLoadersData);
					}
					if (!jsonDeepEquals(currentRiverData, newCurrentRiverData)) {
						setCurrentRiverData(newCurrentRiverData);
					}
					if (outermostErrorIdx !== newOutermostErrorIdx) {
						setOutermostErrorIdx(newOutermostErrorIdx);
					}
					if (outermostError !== newOutermostError) {
						setOutermostError(newOutermostError);
					}
					setLatestEvent(e);
				});
			});
		}
	}, [idx]);

	useLayoutEffect(() => {
		if (!latestEvent || idx !== 0) {
			return;
		}
		window.requestAnimationFrame(() => {
			applyScrollState(latestEvent.detail.scrollState);
		});
	}, [idx, latestEvent]);

	const setLocation = useSetAtom(locationAtom);

	// biome-ignore lint/correctness/useExhaustiveDependencies: nope
	useEffect(() => {
		if (idx === 0) {
			return addLocationListener(() => {
				flushSync(() => {
					setLocation(getLocation());
				});
			});
		}
	}, [idx]);

	useEffect(() => {
		if (idx === 0) {
			return addBuildIDListener((e) => {
				if (!e.detail.fromGETAction) {
					return;
				}
				flushSync(() => {
					setCurrentRiverData(getCurrentRiverData());
				});
			});
		}
	}, [idx]);

	const isErrorIdx = useMemo(() => {
		return idx === outermostErrorIdx;
	}, [idx, outermostErrorIdx]);

	// biome-ignore lint/correctness/useExhaustiveDependencies: nope
	const CurrentComp = useMemo(() => {
		if (isErrorIdx) {
			return null;
		}
		return ctx.get("activeComponents")?.[idx];
	}, [currentImportURL, currentExportKey, isErrorIdx]);

	// biome-ignore lint/correctness/useExhaustiveDependencies: nope
	const Outlet = useMemo(
		() => (localProps: Record<string, any> | undefined) => {
			return <RiverRootOutlet {...localProps} {...props} idx={idx + 1} />;
		},
		[nextImportURL, nextExportKey],
	);

	const shouldFallbackOutletMemo = useMemo(() => {
		if (isErrorIdx) {
			return false;
		}
		if (CurrentComp) {
			return false;
		}
		return idx + 1 < loadersData.length;
	}, [idx, loadersData, isErrorIdx, CurrentComp]);

	const ErrorComp = useMemo(() => {
		if (!isErrorIdx) {
			return null;
		}
		return ctx.get("activeErrorBoundary");
	}, [isErrorIdx]);

	if (isErrorIdx) {
		if (ErrorComp) {
			return <ErrorComp error={outermostError} />;
		}
		return <>{`Error: ${outermostError || "unknown"}`}</>;
	}

	if (!CurrentComp) {
		if (shouldFallbackOutletMemo) {
			return <Outlet />;
		}
		return <></>;
	}

	return <CurrentComp idx={idx} Outlet={Outlet} />;
}
