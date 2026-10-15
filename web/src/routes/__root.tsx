import { Outlet, createRootRouteWithContext } from "@tanstack/react-router";

import { RouteError } from "../app/route-error";
import type { RouterContext } from "../router";
import { Startup } from "../shell/startup/startup";

export const Route = createRootRouteWithContext<RouterContext>()({
  component: Root,
  errorComponent: RouteError,
});

function Root() {
  return (
    <>
      <Startup handOff="fade" />
      <Outlet />
    </>
  );
}
