import { createFileRoute } from "@tanstack/react-router";

import {
  getGetMediaServerPlaybackSuspenseQueryOptions,
  useGetMediaServerSuspense,
} from "../../../../api/gen/query/mediaservers/mediaservers";
import { getListContentTypesSuspenseQueryOptions } from "../../../../api/gen/query/works/works";
import { ensureQuery } from "../../../../api/query-client";
import { PlaybackTab } from "../../../../media-servers/page/playback-tab";
import { serverSectionOf } from "../../../../media-servers/page/sections";
import { Skeleton } from "../../../../ui/skeleton";

const section = serverSectionOf("playback");

export const Route = createFileRoute("/_app/settings/media-servers/$serverId/playback")({
  staticData: { title: section.label },
  loader: ({ context: { queryClient }, params: { serverId } }) =>
    Promise.all([
      ensureQuery(queryClient, getGetMediaServerPlaybackSuspenseQueryOptions(serverId)),
      ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
    ]),
  pendingComponent: () => <Skeleton label="Loading the Playback report" shape="rows" lines={2} />,
  component: function Playback() {
    return <PlaybackTab server={useGetMediaServerSuspense(Route.useParams().serverId).data} />;
  },
});
