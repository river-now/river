import type { RiverRoutes } from "river.now/client";

declare const routes: RiverRoutes;

const rootComp = routes.Component({
	module: import("./components/routes/root.tsx"),
	export: "Root",
});

routes.Register("", rootComp);
routes.Register("/", { module: import("./components/routes/home.tsx"), export: "Home" });
routes.Register("/start", { module: import("./components/routes/start.tsx"), export: "Start" });
routes.Register("/__/:dyn", { module: import("./components/routes/dyn.tsx"), export: "Dyn" });

export default routes;
