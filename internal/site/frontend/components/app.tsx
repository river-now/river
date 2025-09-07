import { done, isStarted, start } from "nprogress";
import { setupGlobalLoadingIndicator } from "river.now/client";
import { setTheme, THEMES } from "river.now/kit/theme";
import { RiverRootOutlet } from "river.now/solid";
import "../styles/tailwind.css";
import { AppLink, theme, useRouterData } from "./app_utils.ts";

setupGlobalLoadingIndicator({ start, stop: done, isRunning: isStarted });

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
			<nav class="w-full flex items-center">
				<div class="flex items-baseline">
					<AppLink pattern="/">
						<h1 class="logo">River</h1>
					</AppLink>
					<div class="text-xs opacity-70 hidden sm:flex">
						({routerData().rootData?.LatestVersion})
					</div>
				</div>

				<div class="flex nav-right">
					<AppLink
						pattern="/*"
						splatValues={["docs"]}
						class="nav-item"
					>
						Docs
					</AppLink>

					<button
						type="button"
						title="Change theme"
						class="cursor-pointer nav-item"
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
							class="w-5 h-5"
						/>
					</button>
				</div>
			</nav>

			<main class="root-outlet-wrapper">
				<RiverRootOutlet />
			</main>

			<footer>
				<span>
					BSD-3-Clause license. Copyright (c) 2023–
					{new Date().getFullYear()} Samuel J. Cook.
				</span>
			</footer>
		</>
	);
}
