import { Outlet, createRootRouteWithContext } from "@tanstack/react-router";

import { RouteError } from "../app/route-error";
import type { RouterContext } from "../router";

export const Route = createRootRouteWithContext<RouterContext>()({
  component: Outlet,
  errorComponent: RouteError,
});
