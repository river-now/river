import { atom, useAtom } from "jotai";
import { type JSX, useEffect, useMemo, useState } from "react";
import { flushSync } from "react-dom";
import {
	addBuildIDListener,
	addRouteChangeListener,
	internal_RiverClientGlobal as ctx,
	getCurrentRiverData,
	type RiverRootOutletPropsGeneric,
} from "river.now/client";
import { jsonDeepEquals } from "river.now/kit/json";

let shouldScroll = false;

const importURLsAtom = atom(ctx.get("importURLs"));
const rootDataAtom = atom(ctx.get("hasRootData") ? ctx.get("loadersData")[0] : null);
const paramsAtom = atom(ctx.get("params") ?? {});
const splatValuesAtom = atom(ctx.get("splatValues") ?? []);
export const loadersDataAtom = atom(ctx.get("loadersData"));
export const currentRiverDataAtom = atom(getCurrentRiverData());

export function RiverRootOutlet(props: RiverRootOutletPropsGeneric<JSX.Element>): JSX.Element {
	const idx = props.idx ?? 0;
	const [currentImportURL, setCurrentImportURL] = useState(ctx.get("importURLs")?.[idx]);
	const [currentExportKey, setCurrentExportKey] = useState(ctx.get("exportKeys")?.[idx]);
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
				if (currentImportURL !== newCurrentImportURL) setCurrentImportURL(newCurrentImportURL);
				if (currentExportKey !== newCurrentExportKey) setCurrentExportKey(newCurrentExportKey);
				if (nextImportURL !== newNextImportURL) setNextImportURL(newNextImportURL);
				if (nextExportKey !== newNextExportKey) setNextExportKey(newNextExportKey);
			});
		});
	}, [currentImportURL, currentExportKey]);

	const [importURLs, setImportURLs] = useAtom(importURLsAtom);
	const [rootData, setRootData] = useAtom(rootDataAtom);
	const [params, setParams] = useAtom(paramsAtom);
	const [splatValues, setSplatValues] = useAtom(splatValuesAtom);
	const [loadersData, setLoadersData] = useAtom(loadersDataAtom);
	const [currentRiverData, setCurrentRiverData] = useAtom(currentRiverDataAtom);

	// biome-ignore lint/correctness/useExhaustiveDependencies: nope
	useEffect(() => {
		if (idx === 0) {
			return addRouteChangeListener((e) => {
				const newImportURLs = ctx.get("importURLs");
				const newRootData = ctx.get(ctx.get("hasRootData") ? ctx.get("loadersData")[0] : null);
				const newParams = ctx.get("params") ?? {};
				const newSplatValues = ctx.get("splatValues") ?? [];
				const newLoadersData = ctx.get("loadersData");
				const newCurrentRiverData = getCurrentRiverData();

				flushSync(() => {
					if (!jsonDeepEquals(importURLs, newImportURLs)) setImportURLs(newImportURLs);
					if (!jsonDeepEquals(rootData, newRootData)) setRootData(newRootData);
					if (!jsonDeepEquals(params, newParams)) setParams(newParams);
					if (!jsonDeepEquals(splatValues, newSplatValues)) setSplatValues(newSplatValues);
					if (!jsonDeepEquals(loadersData, newLoadersData)) setLoadersData(newLoadersData);
					if (!jsonDeepEquals(currentRiverData, newCurrentRiverData))
						setCurrentRiverData(newCurrentRiverData);
				});

				if (e.detail.scrollState) {
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
	}, [idx]);

	// biome-ignore lint/correctness/useExhaustiveDependencies: nope
	useEffect(() => {
		if (idx === 0) {
			return addBuildIDListener((e) => {
				if (!e.detail.fromGETAction) return;
				flushSync(() => {
					setCurrentRiverData(getCurrentRiverData());
				});
			});
		}
	}, [idx]);

	// biome-ignore lint/correctness/useExhaustiveDependencies: nope
	const CurrentComp = useMemo(
		() => ctx.get("activeComponents")?.[idx],
		[currentImportURL, currentExportKey],
	);

	// biome-ignore lint/correctness/useExhaustiveDependencies: nope
	const Outlet = useMemo(
		() => (localProps: Record<string, any> | undefined) => {
			return <RiverRootOutlet {...localProps} {...props} idx={idx + 1} />;
		},
		[nextImportURL, nextExportKey],
	);

	if (!CurrentComp) return <></>;

	return <CurrentComp idx={idx} Outlet={Outlet} />;
}
