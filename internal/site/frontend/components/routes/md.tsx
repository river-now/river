import { hmrRunClientLoaders } from "river.now/client";
import { Show } from "solid-js";
import {
	addClientLoader,
	AppLink,
	useLoaderData,
	type RouteProps,
} from "../app_utils.ts";
import { RenderedMarkdown } from "../rendered-markdown.tsx";

// Use this if you want your client loaders to re-run when you save this file
hmrRunClientLoaders(import.meta);

export const useSplatClientLoaderData = addClientLoader("/*", async (props) => {
	// This is pointless -- just an example of how to use a client loader
	// await new Promise((r) => setTimeout(r, 1_000));
	console.log(`Client loader '/*' started at ${Date.now()}`);
	const { loaderData } = await props.serverDataPromise;
	console.log("Server data promise resolved at ", Date.now(), loaderData);

	// This is how you pass an abort signal to your API calls,
	// so that if the navigation aborts, the downstream requests
	// also abort:
	// const res = await api.mutate({
	// 	pattern: "/example",
	// 	requestInit: { signal: props.signal },
	// });

	return loaderData.Title as string;
});

export function MD(props: RouteProps<"/*">) {
	const loaderData = useLoaderData(props);

	const splatClientLoaderData = useSplatClientLoaderData(props);

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
										<AppLink
											pattern="/*"
											splatValues={["docs"]}
											href={x.url}
										>
											{x.title}
										</AppLink>
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
