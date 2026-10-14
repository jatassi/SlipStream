import { Logout01Icon } from "@hugeicons/core-free-icons";
import { Link, Outlet, createFileRoute } from "@tanstack/react-router";

import { useDeleteSession } from "../api/gen/query/identity/identity";
import { LiveStatus } from "../live/live-status";
import { requireEntry } from "../session/gate";
import { useAccount } from "../session/session";
import { Button } from "../ui/button";
import { Icon } from "../ui/icon";

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
      <header className="flex flex-wrap items-center gap-x-6 gap-y-3">
        <nav aria-label="Main">
          <ul className="flex gap-4">
            <li>
              <Link to="/" className="underline">
                Library
              </Link>
            </li>
            <li>
              <Link to="/activity" className="underline">
                Activity
              </Link>
            </li>
            <li>
              <Link to="/system" className="underline">
                This Node
              </Link>
            </li>
          </ul>
        </nav>
        <LiveStatus />
        <SignOut />
      </header>
      <main>
        <Outlet />
      </main>
    </div>
  );
}

function SignOut() {
  const account = useAccount();
  const { endSession } = Route.useRouteContext();
  const signOut = useDeleteSession({
    mutation: {
      onSuccess: () => {
        endSession("signed-out");
      },
    },
  });
  return (
    <div className="flex items-center gap-3">
      <span className="text-ink-3">{account.displayName}</span>
      <Button
        onClick={() => {
          signOut.mutate();
        }}
        disabled={signOut.isPending}
      >
        <Icon icon={Logout01Icon} />
        Sign out
      </Button>
    </div>
  );
}
