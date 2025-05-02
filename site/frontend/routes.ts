import type { RiverRoutes } from "river.now/client";

declare const routes: RiverRoutes;
export default routes;

routes.Add("/", import("./components/routes/home.tsx"), "Home");
routes.Add("/*", import("./components/routes/md.tsx"), "MD", "ErrorBoundary");
routes.Add("/__/:dyn", import("./components/routes/dyn.tsx"), "Dyn");
