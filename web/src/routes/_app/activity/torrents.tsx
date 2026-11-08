import { createFileRoute, useMatch, useNavigate } from "@tanstack/react-router";

import { loadTorrents, torrentsSearch } from "../../../torrents/load";
import { TorrentsPage } from "../../../torrents/torrents-page";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/activity/torrents")({
  staticData: { title: "Torrents", view: true },
  validateSearch: torrentsSearch,
  loader: ({ context }) => loadTorrents(context.queryClient),
  pendingComponent: () => <Skeleton label="Loading Torrents" />,
  component: ActivityTorrents,
});

function ActivityTorrents() {
  const navigate = useNavigate();
  const selected = useMatch({ from: "/_app/activity/torrents/$torrentId", shouldThrow: false })
    ?.params.torrentId;
  return (
    <TorrentsPage
      search={Route.useSearch()}
      selected={selected}
      onSearchChange={(change) => {
        void navigate({ to: ".", search: (prev) => ({ ...prev, ...change }), replace: true });
      }}
      onSelect={(row) => {
        void navigate({
          to: "/activity/torrents/$torrentId",
          params: { torrentId: row.id },
          search: (prev) => prev,
          replace: selected !== undefined,
        });
      }}
    />
  );
}
