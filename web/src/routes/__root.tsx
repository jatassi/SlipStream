import { HeadContent, Outlet, createRootRouteWithContext } from "@tanstack/react-router";

import { RouteError } from "../app/route-error";
import type { RouterContext } from "../router";
import { Startup } from "../shell/startup/startup";
import { documentTitle } from "../shell/titles";

export const Route = createRootRouteWithContext<RouterContext>()({
  head: ({ matches }) => ({ meta: [{ title: documentTitle(matches) }] }),
  component: Root,
  errorComponent: RouteError,
});

function Root() {
  return (
    <>
      <HeadContent />
      <Startup handOff="fade" />
      <Outlet />
    </>
  );
}
