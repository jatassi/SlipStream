import { Outlet, createFileRoute } from "@tanstack/react-router";

import { LiveStatus } from "../live/live-status";
import { requireEntry } from "../session/gate";

export const Route = createFileRoute("/_app")({
  beforeLoad: async ({ context, location }) => {
    await requireEntry(context, "app", location);
    await context.live.connect();
  },
  component: AppLayout,
});

function AppLayout() {
  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-6 px-4 py-6">
      <LiveStatus />
      <main>
        <Outlet />
      </main>
    </div>
  );
}
