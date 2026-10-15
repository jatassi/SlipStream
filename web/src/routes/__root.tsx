import "../shell/motion.css";

import { HeadContent, Outlet, createRootRouteWithContext, useMatch } from "@tanstack/react-router";
import { useEffect } from "react";

import { RouteError } from "../app/route-error";
import type { RouterContext } from "../router";
import { Ambient } from "../shell/ambient/ambient";
import { densityOf, useShellKind } from "../shell/media";
import { Startup } from "../shell/startup/startup";
import { documentTitle } from "../shell/titles";
import { ShellContext } from "../ui/shell-context";
import { Toasts } from "../ui/toasts";

export const Route = createRootRouteWithContext<RouterContext>()({
  head: ({ matches }) => ({ meta: [{ title: documentTitle(matches) }] }),
  component: Root,
  errorComponent: RouteError,
});

function Root() {
  const shell = useShellKind();
  const inApp = useMatch({ from: "/_app", shouldThrow: false }) !== undefined;
  // Kit popups portal into body, outside any shell root, so the density sits on <html>.
  useEffect(() => {
    document.documentElement.dataset["density"] = densityOf[shell];
  }, [shell]);
  return (
    <ShellContext value={shell}>
      <HeadContent />
      <Ambient />
      <Toasts />
      <Startup handOff={shell === "desktop" && inApp ? "view-transition" : "fade"} />
      <Outlet />
    </ShellContext>
  );
}
