import { hmrRunClientLoaders } from "river.now/client";
import { Show } from "solid-js";
import { Link } from "../app_link.tsx";
import {
	addClientLoader,
	type RouteProps,
	useLoaderData,
} from "../app_utils.ts";
import { RenderedMarkdown } from "../rendered-markdown.tsx";

// Use this if you want your client loaders to re-run when you save this file
hmrRunClientLoaders(import.meta);

const useClientLoaderData = addClientLoader("/*", async (props) => {
	// This is pointless -- just an example of how to use a client loader
	// await new Promise((r) => setTimeout(r, 1_000));
	console.log("Client loader running");
	return props.loaderData.Title;
});

export type CatchAllCLD = ReturnType<typeof useClientLoaderData>;

export function MD(props: RouteProps<"/*">) {
	const loaderData = useLoaderData(props);
	const clientLoaderData = useClientLoaderData(props);

	return (
		<div class="content">
			<Show when={clientLoaderData()}>{(n) => <h1>{n()}</h1>}</Show>
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
										<Link prefetch="intent" href={x.url}>
											{x.title}
										</Link>
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
