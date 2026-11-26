import { createFileRoute } from "@tanstack/react-router";

import { loadMyAccess } from "../../media-servers/access/portal";
import { WatchCard } from "../../media-servers/access/watch-card";
import { useAccount } from "../../session/session";
import { Page } from "../../shell/page";

export const Route = createFileRoute("/portal/")({
  staticData: { title: "Home" },
  loader: ({ context }) => loadMyAccess(context.queryClient),
  component: Home,
});

function Home() {
  const { admin } = useAccount();
  return (
    <Page>
      <div className="flex flex-col gap-6">
        {!admin && <WatchCard />}
        <p className="text-body-touch text-ink-2">
          New arrivals and the things you&apos;ve asked for will show here.
        </p>
      </div>
    </Page>
  );
}
