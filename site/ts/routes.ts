import type { RiverRoutes } from "@sjc5/river/client";

declare const routes: RiverRoutes;

const rootComp = routes.Component({ module: import("./root.tsx"), export: "Root" });

routes.Register("", rootComp);
routes.Register("/", { module: import("./app.tsx"), export: "Home" });

export default routes;
