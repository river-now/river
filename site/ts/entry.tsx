import { getRootEl, initClient } from "@sjc5/river/client";
import { render } from "solid-js/web";
import { App } from "./app.tsx";

await initClient(() => {
	render(() => <App />, getRootEl());
});
