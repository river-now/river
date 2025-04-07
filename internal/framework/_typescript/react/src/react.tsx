import {
	addRouteChangeListener,
	internal_RiverClientGlobal as ctx,
	getCurrentRiverData,
	type RootOutletProps,
} from "@sjc5/river/client";
import { jsonDeepEquals } from "@sjc5/river/kit/json";
import { atom, useAtom } from "jotai";
import { type JSX, useEffect, useMemo, useState } from "react";
import { flushSync } from "react-dom";

let shouldScroll = false;

const importURLsAtom = atom(ctx.get("importURLs"));
const coreDataAtom = atom(ctx.get("coreData"));
const paramsAtom = atom(ctx.get("params") ?? {});
const splatValuesAtom = atom(ctx.get("splatValues") ?? []);
export const loadersDataAtom = atom(ctx.get("loadersData"));
export const currentRiverDataAtom = atom(getCurrentRiverData());

export function RiverRootOutlet(props: RootOutletProps<JSX.Element>): JSX.Element {
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
	const [coreData, setCoreData] = useAtom(coreDataAtom);
	const [params, setParams] = useAtom(paramsAtom);
	const [splatValues, setSplatValues] = useAtom(splatValuesAtom);
	const [loadersData, setLoadersData] = useAtom(loadersDataAtom);
	const [currentRiverData, setCurrentRiverData] = useAtom(currentRiverDataAtom);

	// biome-ignore lint/correctness/useExhaustiveDependencies: nope
	useEffect(() => {
		if (idx === 0) {
			return addRouteChangeListener((e) => {
				const newImportURLs = ctx.get("importURLs");
				const newCoreData = ctx.get("coreData");
				const newParams = ctx.get("params") ?? {};
				const newSplatValues = ctx.get("splatValues") ?? [];
				const newLoadersData = ctx.get("loadersData");
				const newCurrentRiverData = getCurrentRiverData();

				flushSync(() => {
					if (!jsonDeepEquals(importURLs, newImportURLs)) setImportURLs(newImportURLs);
					if (!jsonDeepEquals(coreData, newCoreData)) setCoreData(newCoreData);
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
