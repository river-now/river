import type { RiverRoutes } from "river.now/client";

declare const routes: RiverRoutes;
export default routes;

/********************************************************************
  SPECIAL RULES FOR THIS FILE:
    1. Do not import anything other than types.
    2. Do not rename the `routes` variable.
    3. Always use the true extension for import paths (e.g., ".tsx")
********************************************************************/

routes.Add("/_index", import("./home.tsx"), "Home");
