import { createFileRoute, useNavigate } from "@tanstack/react-router";

import { loaderTitle } from "../../../../../shell/titles";
import { InspectedTorrent, loadInspected } from "../../../../../torrents/inspected";
import { Skeleton } from "../../../../../ui/skeleton";

export const Route = createFileRoute(
  "/_app/settings/trackers/$trackerId/_torrents/torrents/$torrentId",
)({
  staticData: { title: loaderTitle },
  loader: ({ context: { queryClient }, params: { torrentId } }) =>
    loadInspected(queryClient, torrentId),
  pendingComponent: () => <Skeleton label="Loading the torrent" />,
  component: function TrackerTorrent() {
    const navigate = useNavigate();
    const { trackerId, torrentId } = Route.useParams();
    return (
      <InspectedTorrent
        torrentId={torrentId}
        onClose={(gone) => {
          void navigate({
            to: "/settings/trackers/$trackerId",
            params: { trackerId },
            search: (prev) => prev,
            replace: gone,
          });
        }}
      />
    );
  },
});
