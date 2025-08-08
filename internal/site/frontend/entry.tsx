import { getRootEl, initClient } from "river.now/client";
import { render } from "solid-js/web";
import { App } from "./components/app.tsx";

await initClient(
	() => {
		render(() => <App />, getRootEl());
	},
	{ useViewTransitions: false, defaultErrorBoundary: undefined },
);

import("./components/highlight.ts"); // warm up highlighter
