import { getRootEl, initClient } from "river.now/client";
import { render } from "solid-js/web";
import { App } from "./components/app.tsx";
import { riverAppConfig } from "./river.gen.ts";

await initClient(
	() => {
		render(() => <App />, getRootEl());
	},
	{
		riverAppConfig,
		useViewTransitions: false,
		defaultErrorBoundary: undefined,
	},
);

import("./components/highlight.ts"); // warm up highlighter
