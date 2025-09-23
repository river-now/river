import type { RiverRoutes } from "river.now/client";

declare const routes: RiverRoutes;
export default routes;

routes.Add("/", import("./components/home.tsx"), "RootLayout");
routes.Add("/_index", import("./components/home.tsx"), "Home");
routes.Add("/*", import("./components/md.tsx"), "MD", "ErrorBoundary");
routes.Add("/__/:dyn", import("./components/dyn.tsx"), "Dyn");
