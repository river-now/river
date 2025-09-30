import { getRootEl, initClient } from "river.now/client";
import { render } from "solid-js/web";
import { App } from "./components/app.tsx";
import { riverAppConfig } from "./river.gen.ts";

await initClient({
	riverAppConfig,
	renderFn: () => {
		render(() => <App />, getRootEl());
	},
});

import("./highlight.ts"); // warm up highlighter
import("./html_to_md.ts"); // warm up markdown converter
import("./components/md.tsx"); // warm up  markdown route component
