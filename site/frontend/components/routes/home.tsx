import { usePatternClientLoaderData } from "river.now/solid";
import { Link } from "../app_link.tsx";
import {
	addClientLoader,
	type RouteProps,
	usePatternLoaderData,
} from "../app_utils.ts";

const useClientLoaderData = addClientLoader("", async (props) => {
	// This is pointless -- just an example of how to use a client loader
	// await new Promise((r) => setTimeout(r, 1_000));
	console.log("Client loader running");
	return props.loaderData.LatestVersion;
});

type RootCLD = ReturnType<typeof useClientLoaderData>;

export function Home(props: RouteProps<"/">) {
	const x = usePatternLoaderData("");
	const y = usePatternClientLoaderData<RootCLD>("");
	// console.log("x", x());
	// console.log("y", y());

	return (
		<div class="flex h-full justify-center flex-col items-center mb-16 gap-6">
			<h2 class="text-center text-balance text-3xl sm:text-4xl md:text-5xl lg:text-5xl w-5xl max-w-full leading-normal my-6 px-6">
				River is a <FancySpan>Go</FancySpan> / <FancySpan>TypeScript</FancySpan>{" "}
				meta-framework with first-class support for <FancySpan>React</FancySpan>,{" "}
				<FancySpan>Solid</FancySpan>, and <FancySpan>Preact</FancySpan> – built on{" "}
				<FancySpan>Vite</FancySpan>.
			</h2>

			<Link
				href="/start"
				type="button"
				class="py-4 px-6 bg-[var(--dark-green)] dark:bg-[var(--light-green)] text-white dark:text-black rounded-lg shadow-md hover:opacity-80 hover:outline-2 outline-black dark:outline-white outline-offset-2 active:opacity-100 cursor-pointer uppercase tracking-wider font-bold"
			>
				Get Started
			</Link>
		</div>
	);
}

function FancySpan(props: { children: string }) {
	return (
		<span class="font-bold italic text-[#064929] dark:text-[#4BBA5B]">
			{props.children}
		</span>
	);
}
