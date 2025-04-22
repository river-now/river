import { Show } from "solid-js";
import { Link } from "../app_link.tsx";
import { type RouteProps, useLoaderData } from "../app_utils.ts";
import { RenderedMarkdown } from "../rendered-markdown.tsx";

// preload("/*", async (props) => {
// 	console.log("Loading MD route...", props);
// 	await new Promise((resolve) => setTimeout(resolve, 300));
// });

export function MD(props: RouteProps<"/*">) {
	return (
		<div class="content">
			<Show when={useLoaderData(props)?.Title}>{(n) => <h1>{n()}</h1>}</Show>
			<Show when={useLoaderData(props)?.Date}>{(n) => <i>{n()}</i>}</Show>
			<Show when={useLoaderData(props)?.Content}>
				{(n) => <RenderedMarkdown markdown={n()} />}
			</Show>
			<Show when={useLoaderData(props)?.IndexSitemap}>
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
