import { createRoot } from "react-dom/client";
import { getRootEl, initClient } from "river.now/client";
import { RiverProvider } from "river.now/react";
import { App } from "./app.tsx";

await initClient(() => {
	createRoot(getRootEl()).render(
		<RiverProvider>
			<App />
		</RiverProvider>,
	);
});
