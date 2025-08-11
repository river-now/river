import { usePatternClientLoaderData } from "river.now/solid";
import { Link } from "../app_link.tsx";
import {
	addClientLoader,
	type RouteProps,
	usePatternLoaderData,
} from "../app_utils.ts";

const useClientLoaderData = addClientLoader("/", async (props) => {
	// This is pointless -- just an example of how to use a client loader
	// await new Promise((r) => setTimeout(r, 1_000));
	console.log("Client loader running");
	return props.loaderData.LatestVersion;
});

type RootCLD = ReturnType<typeof useClientLoaderData>;

export function Home(_props: RouteProps<"/_index">) {
	const _x = usePatternLoaderData("");
	const _y = usePatternClientLoaderData<RootCLD>("");
	// console.log("_x", _x());
	// console.log("_y", _y());

	return (
		<>
			<h1 class="big-heading">
				River is a{" "}
				<b>
					<i>simple</i>
				</b>
				,{" "}
				<b>
					<i>lightweight</i>
				</b>
				, and{" "}
				<b>
					<i>flexible</i>
				</b>{" "}
				web framework for{" "}
				<span class="whitespace-nowrap">
					<b>
						<i>Go</i>
					</b>
					<span class="p-[2px] font-extralight">/</span>
					<b>
						<i>TypeScript</i>
					</b>
				</span>
				, built on{" "}
				<b>
					<i>Vite</i>
				</b>
				.
			</h1>

			<div class="flex gap-3 flex-wrap mb-6">
				<a
					class="font-medium bg-[var(--fg)] py-[2px] px-[6px] text-[var(--bg)] text-sm rounded-sm cursor-pointer hover:bg-blue-700 dark:hover:bg-blue-200"
					href="https://github.com/river-now/river"
					target="_blank"
					rel="noreferrer"
				>
					⭐ github.com
				</a>

				<a
					class="font-medium bg-[var(--fg)] py-[2px] px-[6px] text-[var(--bg)] text-sm rounded-sm cursor-pointer hover:bg-blue-700 dark:hover:bg-blue-200"
					href="https://pkg.go.dev/github.com/river-now/river"
					target="_blank"
					rel="noreferrer"
				>
					🔷 pkg.go.dev
				</a>

				<a
					class="font-medium bg-[var(--fg)] py-[2px] px-[6px] text-[var(--bg)] text-sm rounded-sm cursor-pointer hover:bg-blue-700 dark:hover:bg-blue-200"
					href="https://www.npmjs.com/package/river.now"
					target="_blank"
					rel="noreferrer"
				>
					📦 npmjs.com
				</a>

				<a
					class="font-medium bg-[var(--fg)] py-[2px] px-[6px] text-[var(--bg)] text-sm rounded-sm cursor-pointer hover:bg-blue-700 dark:hover:bg-blue-200"
					href="https://x.com/riverframework"
					target="_blank"
					rel="noreferrer"
				>
					𝕏 x.com
				</a>
			</div>

			<div>
				<h3 class="h3">Quickstart</h3>
				<code class="inline-code high-contrast self-start text-xl font-bold italic">
					npm create river@latest
				</code>
			</div>

			<div>
				<h3 class="h3">What is River?</h3>
				<div class="flex-col-wrapper">
					<p class="leading-[1.75]">
						River is a lot like NextJS or Remix, but it uses{" "}
						<b>
							<i>Go</i>
						</b>{" "}
						on the backend, with your choice of{" "}
						<b>
							<i>React</i>
						</b>
						,{" "}
						<b>
							<i>Solid</i>
						</b>
						, or{" "}
						<b>
							<i>Preact</i>
						</b>{" "}
						on the frontend.
					</p>

					<p class="leading-[1.75]">
						It has{" "}
						<b>
							<i>nested routing</i>
						</b>
						,{" "}
						<b>
							<i>type-safe server actions</i>
						</b>
						, and{" "}
						<b>
							<i>parallel-executed route loaders</i>
						</b>
						.
					</p>

					<p class="leading-[1.75]">
						And it's deeply integrated with{" "}
						<b>
							<i>Vite</i>
						</b>{" "}
						to give you{" "}
						<b>
							<i>hot module reloading</i>
						</b>{" "}
						at dev-time.
					</p>
				</div>
			</div>

			<div>
				<h3 class="h3">Get started</h3>
				<div class="flex-col-wrapper">
					<p class="leading-[1.75]">
						If you want to dive right in, just open a terminal and
						run{" "}
						<code class="inline-code">npm create river@latest</code>{" "}
						and follow the prompts.
					</p>
					<p class="leading-[1.75]">
						If you'd prefer to read more first, take a peek at{" "}
						<Link href="/docs" class="underline">
							our docs
						</Link>
						.
					</p>
				</div>
			</div>

			<div>
				<h3 class="h3">Disclaimer</h3>
				<p class="leading-[1.75]">
					River is in beta! Act accordingly. It has, however, been
					used in anger for quite some time, and it is not too far
					from a stable 1.0 release. It's a perfect time to give River
					a try for your next side project or internal tool.
				</p>
			</div>
		</>
	);
}
