import type { RiverRoutes } from "river.now/client";

declare const routes: RiverRoutes;
export default routes;

const r = routes.New;

r("", import("./components/routes/root.tsx"), "Root");
r("/", import("./components/routes/home.tsx"), "Home");
r("/*", import("./components/routes/md.tsx"), "MD", "ErrorBoundary");
r("/__/:dyn", import("./components/routes/dyn.tsx"), "Dyn");
