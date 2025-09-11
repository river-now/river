import { RiverLink } from "river.now/solid";
import { Show } from "solid-js";
import {
	addClientLoader,
	useLoaderData,
	type RouteProps,
} from "../app_utils.ts";
import { RenderedMarkdown } from "../rendered-markdown.tsx";
import { useRootClientLoaderData } from "./home.tsx";

export const useSplatClientLoaderData = addClientLoader({
	pattern: "/*",
	clientLoader: async (props) => {
		// This is pointless -- just an example of how to use a client loader
		// await new Promise((r) => setTimeout(r, 1_000));
		// console.log(`Client loader '/*' started at ${Date.now()}`);
		const { loaderData } = await props.serverDataPromise;
		// console.log("Server data promise resolved at ", Date.now(), loaderData);

		// This is how you pass an abort signal to your API calls,
		// so that if the navigation aborts, the downstream requests
		// also abort:
		// const res = await api.mutate({
		// 	pattern: "/example",
		// 	requestInit: { signal: props.signal },
		// });

		console.log("MD.TSX CLIENT LOADER");

		return loaderData.Title as string;
	},
	reRunOnModuleChange: import.meta,
});

export function MD(props: RouteProps<"/*">) {
	const loaderData = useLoaderData(props);

	const splatClientLoaderData = useSplatClientLoaderData(props);
	const _y = useRootClientLoaderData();
	// console.log("_y", _y());

	return (
		<div class="content">
			<Show when={splatClientLoaderData()}>{(n) => <h1>{n()}</h1>}</Show>
			<Show when={loaderData()?.Date}>{(n) => <i>{n()}</i>}</Show>
			<Show when={loaderData()?.Content}>
				{(n) => <RenderedMarkdown markdown={n()} />}
			</Show>
			<Show when={loaderData()?.IndexSitemap}>
				{(n) => (
					<div class={"content"}>
						<ul>
							{n().map((x) => {
								return (
									<li>
										<RiverLink
											href={x.url}
											prefetch="intent"
										>
											{x.title}
										</RiverLink>
									</li>
								);
							})}
						</ul>
					</div>
				)}
			</Show>
		</div>
	);
}

export function ErrorBoundary(props: { error: string }) {
	return <div>Error: {props.error}</div>;
}
