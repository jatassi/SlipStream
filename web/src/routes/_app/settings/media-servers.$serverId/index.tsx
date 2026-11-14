import { createFileRoute } from "@tanstack/react-router";

import {
  getListMediaServerSessionsSuspenseQueryOptions,
  useGetMediaServerSuspense,
} from "../../../../api/gen/query/mediaservers/mediaservers";
import { ensureQuery } from "../../../../api/query-client";
import { PlayingTab } from "../../../../media-servers/page/playing-tab";
import { loaderTitle } from "../../../../shell/titles";
import { Skeleton } from "../../../../ui/skeleton";

export const Route = createFileRoute("/_app/settings/media-servers/$serverId/")({
  staticData: { title: loaderTitle },
  loader: ({ context: { queryClient }, params: { serverId } }) =>
    ensureQuery(queryClient, getListMediaServerSessionsSuspenseQueryOptions(serverId)),
  pendingComponent: () => <Skeleton label="Loading what's playing" />,
  component: function NowPlaying() {
    return <PlayingTab server={useGetMediaServerSuspense(Route.useParams().serverId).data} />;
  },
});
