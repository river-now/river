import type { RiverRoutes } from "@sjc5/river/client";

declare const routes: RiverRoutes;

const rootComp = routes.Component({ module: import("./root.tsx"), export: "Root" });

routes.Register("", rootComp);
routes.Register("/", { module: import("./home.tsx"), export: "Home" });
routes.Register("/start", { module: import("./start.tsx"), export: "Start" });

export default routes;
