import { Outlet, createFileRoute } from "@tanstack/react-router";
import type { ComponentType, ReactNode } from "react";

import { requireEntry } from "../session/gate";
import { DesktopShell } from "../shell/desktop/layout";
import { Palette } from "../shell/palette/palette";
import { PhoneShell } from "../shell/phone/layout";
import { type ShellKind, useShell } from "../ui/shell-context";

export const Route = createFileRoute("/_app")({
  beforeLoad: async ({ context, location }) => {
    await requireEntry(context, "app", location);
    await context.live.connect();
  },
  component: AppLayout,
});

const shells: { readonly [S in ShellKind]: ComponentType<{ children: ReactNode }> } = {
  desktop: DesktopShell,
  phone: PhoneShell,
};

function AppLayout() {
  const Shell = shells[useShell()];
  // The shell's bars stack inside this context, so popups that portal into
  // body still open above them.
  return (
    <div className="isolate">
      <Shell>
        <Outlet />
      </Shell>
      <Palette />
    </div>
  );
}
