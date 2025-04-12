import { addBuildIDListener, getRootEl, initClient } from "@sjc5/river/client";
import { render } from "solid-js/web";
import { App } from "./home.tsx";

await initClient(() => {
	render(() => <App />, getRootEl());
});

addBuildIDListener((e) => {
	window.location.reload();
});
