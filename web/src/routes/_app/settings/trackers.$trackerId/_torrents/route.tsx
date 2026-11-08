import { createFileRoute, useMatch, useNavigate } from "@tanstack/react-router";

import { getListTrackersSuspenseQueryOptions } from "../../../../../api/gen/query/trackers/trackers";
import { ensureQuery } from "../../../../../api/query-client";
import { loaderTitle } from "../../../../../shell/titles";
import { loadTorrents, torrentsSearch } from "../../../../../torrents/load";
import { TorrentsPage } from "../../../../../torrents/torrents-page";
import { trackerOf, useTracker } from "../../../../../trackers/tracker";
import { Empty } from "../../../../../shell/empty";
import { Skeleton } from "../../../../../ui/skeleton";

export const Route = createFileRoute("/_app/settings/trackers/$trackerId/_torrents")({
  staticData: { title: loaderTitle },
  validateSearch: torrentsSearch,
  loader: async ({ context: { queryClient }, params: { trackerId } }) => {
    const [{ trackers }] = await Promise.all([
      ensureQuery(queryClient, getListTrackersSuspenseQueryOptions()),
      loadTorrents(queryClient),
    ]);
    return { title: trackerOf(trackers, trackerId).name };
  },
  pendingComponent: () => <Skeleton label="Loading the torrents" />,
  component: TrackerTorrents,
});

function TrackerTorrents() {
  const navigate = useNavigate();
  const { trackerId } = Route.useParams();
  const tracker = useTracker(trackerId);
  const selected = useMatch({
    from: "/_app/settings/trackers/$trackerId/_torrents/torrents/$torrentId",
    shouldThrow: false,
  })?.params.torrentId;
  return (
    <TorrentsPage
      search={Route.useSearch()}
      selected={selected}
      scope={{
        keep: (row) => row.tracker?.id === trackerId,
        empty: <Empty>SlipStream has no torrents from {tracker.name}.</Empty>,
      }}
      onSearchChange={(change) => {
        void navigate({ to: ".", search: (prev) => ({ ...prev, ...change }), replace: true });
      }}
      onSelect={(row) => {
        void navigate({
          to: "/settings/trackers/$trackerId/torrents/$torrentId",
          params: { trackerId, torrentId: row.id },
          search: (prev) => prev,
          replace: selected !== undefined,
        });
      }}
    />
  );
}
