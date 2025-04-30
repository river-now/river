import { hmrRunClientLoaders } from "river.now/client";
import { Show } from "solid-js";
import { Link } from "../app_link.tsx";
import { addClientLoader, type RouteProps, useLoaderData } from "../app_utils.ts";
import { RenderedMarkdown } from "../rendered-markdown.tsx";

// hmrRunClientLoaders(import.meta);

// export const useClientLoaderData = addClientLoader("/*", async (props) => {
// 	await new Promise((resolve) => setTimeout(resolve, 300));
// 	return "jeff2";
// });

export function MD(props: RouteProps<"/*">) {
	const loaderData = useLoaderData(props);
	// const clientLoaderData = useClientLoaderData(props);

	return (
		<div class="content">
			{/* Client loader data: {clientLoaderData()} */}
			<Show when={loaderData()?.Title}>{(n) => <h1>{n()}</h1>}</Show>
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

export function ErrorBoundary() {
	return null;
}
