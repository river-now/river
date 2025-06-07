import nProgress from "nprogress";
import { setupGlobalLoadingIndicator } from "river.now/client";
import { setTheme, THEMES } from "river.now/kit/theme";
import { RiverRootOutlet } from "river.now/solid";
import "../css/tailwind-output.css";
import { Link } from "./app_link.tsx";
import { theme, useRouterData } from "./app_utils.ts";

setupGlobalLoadingIndicator({
	include: "all",
	start: nProgress.start,
	stop: nProgress.done,
	isRunning: nProgress.isStarted,
});

const theme_to_label_map = {
	[THEMES.Light]: hashedURL("sun.svg"),
	[THEMES.Dark]: hashedURL("moon.svg"),
	[THEMES.System]: hashedURL("desktop.svg"),
};

for (const url of Object.values(theme_to_label_map)) {
	preload_img(url);
}

// __TODO move to kit
function preload_img(url: string) {
	const img = new Image();
	img.src = url;
	return new Promise((resolve, reject) => {
		img.onload = resolve;
		img.onerror = reject;
	});
}

export function App() {
	const routerData = useRouterData();

	return (
		<>
			<nav class="w-full sticky top-0 bg-white dark:bg-[#111] z-50">
				<div class="flex items-center max-w-full mx-auto p-4 border-b border-[#7777]">
					<h1 class="flex">
						<Link href="/" class="inline-flex items-center gap-3">
							<img
								class="w-10 sm:w-14 aspect-2/1"
								src={hashedURL("logo.svg")}
								alt="River logo"
							/>
							<div class="flex gap-2 items-baseline">
								<div class="sm:text-xl">River</div>
								<div class="text-xs opacity-70 hidden sm:block">
									({routerData().rootData?.LatestVersion})
								</div>
							</div>
						</Link>
					</h1>

					<button
						type="button"
						title="Change theme"
						class="cursor-pointer ml-auto"
						onClick={() => {
							if (theme() === "dark") {
								setTheme(THEMES.Light);
								return;
							}
							if (theme() === "light") {
								setTheme(THEMES.System);
								return;
							}
							if (theme() === "system") {
								setTheme(THEMES.Dark);
								return;
							}
						}}
					>
						<img
							src={theme_to_label_map[theme()]}
							alt="Theme icon"
							class="w-5 h-5 dark:invert"
						/>
					</button>
				</div>
				<div class="flex items-center max-w-full mx-auto px-4 py-1 border-b border-[#7777] gap-4">
					<Link
						href="/start"
						class="text-sm opacity-70 hover:opacity-[unset] hover:underline"
					>
						Get Started
					</Link>

					<Link
						href="/faq"
						class="text-sm opacity-70 hover:opacity-[unset] hover:underline"
					>
						FAQs
					</Link>

					<div class="flex-1" />
					<a
						href="https://x.com/riverframework"
						class="text-sm opacity-70 hover:opacity-[unset] hover:underline"
						target="_blank"
						rel="noreferrer"
					>
						X
					</a>

					<a
						href="https://github.com/river-now/river"
						class="text-sm opacity-70 hover:opacity-[unset] hover:underline"
						target="_blank"
						rel="noreferrer"
					>
						GitHub
					</a>
				</div>
			</nav>

			<main class="z-1">
				<RiverRootOutlet />
			</main>
		</>
	);
}
