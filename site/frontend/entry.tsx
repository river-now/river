import { getRootEl, initClient } from "river.now/client";
import { render } from "solid-js/web";
import { App } from "./components/routes/home.tsx";

await initClient(() => {
	render(() => <App />, getRootEl());
});

import("./components/highlight.ts"); // warm up highlighter
