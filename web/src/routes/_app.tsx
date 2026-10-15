import { Outlet, createFileRoute } from "@tanstack/react-router";

import { requireEntry } from "../session/gate";
import { DesktopShell } from "../shell/desktop/layout";

export const Route = createFileRoute("/_app")({
  beforeLoad: async ({ context, location }) => {
    await requireEntry(context, "app", location);
    await context.live.connect();
  },
  component: AppLayout,
});

function AppLayout() {
  return (
    <DesktopShell>
      <Outlet />
    </DesktopShell>
  );
}
