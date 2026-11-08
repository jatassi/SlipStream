import { createFileRoute, useNavigate } from "@tanstack/react-router";

import { loaderTitle } from "../../../shell/titles";
import { InspectedTorrent, loadInspected } from "../../../torrents/inspected";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/activity/torrents/$torrentId")({
  staticData: { title: loaderTitle },
  loader: ({ context: { queryClient }, params: { torrentId } }) =>
    loadInspected(queryClient, torrentId),
  pendingComponent: () => <Skeleton label="Loading the torrent" />,
  component: function TorrentPage() {
    const navigate = useNavigate();
    return (
      <InspectedTorrent
        torrentId={Route.useParams().torrentId}
        onClose={(gone) => {
          void navigate({ to: "/activity/torrents", search: (prev) => prev, replace: gone });
        }}
      />
    );
  },
});
