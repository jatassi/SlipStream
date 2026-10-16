import "../shell/motion.css";

import { HeadContent, Outlet, createRootRouteWithContext, useMatch } from "@tanstack/react-router";
import { useEffect } from "react";

import type { RouterContext } from "../router";
import { Ambient } from "../shell/ambient/ambient";
import { RootNotFound, ShellError } from "../shell/errors";
import { densityOf, useShellKind } from "../shell/media";
import { Startup } from "../shell/startup/startup";
import { documentTitle } from "../shell/titles";
import { DensityContext, ShellContext } from "../ui/shell-context";
import { Toasts } from "../ui/toasts";

export const Route = createRootRouteWithContext<RouterContext>()({
  head: ({ matches }) => ({ meta: [{ title: documentTitle(matches) }] }),
  component: Root,
  errorComponent: ShellError,
  notFoundComponent: RootNotFound,
});

function Root() {
  const shell = useShellKind();
  const inApp = useMatch({ from: "/_app", shouldThrow: false }) !== undefined;
  const inPortal = useMatch({ from: "/portal", shouldThrow: false }) !== undefined;
  const density = inPortal ? "touch" : densityOf[shell];
  // Kit popups portal into body, outside any shell root, so the density sits on <html>.
  useEffect(() => {
    document.documentElement.dataset["density"] = density;
  }, [density]);
  return (
    <ShellContext value={shell}>
      <DensityContext value={density}>
        <HeadContent />
        <Ambient />
        <Toasts />
        <Startup handOff={shell === "desktop" && inApp ? "view-transition" : "fade"} />
        <Outlet />
      </DensityContext>
    </ShellContext>
  );
}
