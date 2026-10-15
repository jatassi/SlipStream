import { Outlet, createFileRoute } from "@tanstack/react-router";
import type { ComponentType, ReactNode } from "react";

import { requireEntry } from "../session/gate";
import { NotFound, ShellError } from "../shell/errors";
import { portalNav, portalRows } from "../shell/nav";
import { PhoneFrame } from "../shell/phone/frame";
import { PortalDesktop } from "../shell/portal/desktop";
import { type ShellKind, useShell } from "../ui/shell-context";

export const Route = createFileRoute("/portal")({
  beforeLoad: async ({ context, location }) => {
    await requireEntry(context, "portal", location);
    await context.live.connect();
  },
  component: () => (
    <PortalFrame>
      <Outlet />
    </PortalFrame>
  ),
  errorComponent: ShellError,
  notFoundComponent: () => (
    <PortalFrame>
      <NotFound home={portalNav[0]} />
    </PortalFrame>
  ),
});

function PortalPhone({ children }: { children: ReactNode }) {
  return (
    <PhoneFrame rows={portalRows} barActions={null} tabActions={[]}>
      {children}
    </PhoneFrame>
  );
}

const shells: { readonly [S in ShellKind]: ComponentType<{ children: ReactNode }> } = {
  desktop: PortalDesktop,
  phone: PortalPhone,
};

function PortalFrame({ children }: { children: ReactNode }) {
  const Shell = shells[useShell()];
  return (
    <div className="isolate">
      <Shell>{children}</Shell>
    </div>
  );
}
