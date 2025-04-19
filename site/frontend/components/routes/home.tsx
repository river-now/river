import { RiverRootOutlet } from "river.now/solid";
import "../../css/tailwind-output.css";
import { Link } from "../app_link.tsx";
import type { RouteProps } from "../app_utils.ts";

export function App() {
	return <RiverRootOutlet />;
}

export function Home(props: RouteProps<"/">) {
	return (
		<div class="flex h-full justify-center flex-col items-center mb-16 gap-6">
			<h2 class="text-center text-3xl sm:text-5xl w-3xl max-w-full leading-normal my-6">
				River is a framework for writing <FancySpan>modern</FancySpan>,{" "}
				<FancySpan>type-safe</FancySpan> web applications with <FancySpan>Go</FancySpan> and{" "}
				<FancySpan>TypeScript</FancySpan>.
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
	return <span class="font-bold italic text-[#064929] dark:text-[#4BBA5B]">{props.children}</span>;
}
