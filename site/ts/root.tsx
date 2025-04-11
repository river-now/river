import { setTheme, THEMES } from "@sjc5/river/kit/theme";
import { RiverLink } from "@sjc5/river/solid";
import { type RouteProps, theme, useCurrentAppData } from "./app_utils.ts";

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

export function Root(props: RouteProps<"">) {
	return (
		<>
			<nav class="flex items-center max-w-full mx-auto p-4 border-b border-[#7777] fixed top-0 left-0 w-full">
				<h1 class="flex">
					<RiverLink href="/" class="inline-flex items-center gap-4">
						<img class="w-18" src={hashedURL("logo.svg")} alt="River logo" />
						<span class="text-2xl">{useCurrentAppData().rootData}</span>
					</RiverLink>
				</h1>

				<button
					type="button"
					title="Change theme"
					class="cursor-pointer ml-auto text-3xl"
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
					<img src={theme_to_label_map[theme()]} alt="Theme icon" class="w-6 h-6 dark:invert" />
				</button>
			</nav>

			<main class="mt-18 p-4">
				<props.Outlet />
			</main>
		</>
	);
}
