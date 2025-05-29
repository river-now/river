import { render } from "preact";
import { getRootEl, initClient } from "river.now/client";
import { App } from "./app.tsx";

await initClient(() => {
	render(<App />, getRootEl());
});
