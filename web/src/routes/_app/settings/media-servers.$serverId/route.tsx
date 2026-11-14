import { createFileRoute } from "@tanstack/react-router";

import {
  getGetMediaServerSuspenseQueryOptions,
  getListMediaServerKindsSuspenseQueryOptions,
} from "../../../../api/gen/query/mediaservers/mediaservers";
import { ensureQuery } from "../../../../api/query-client";
import { ServerLayout } from "../../../../media-servers/page/layout";
import { loaderTitle } from "../../../../shell/titles";
import { Skeleton } from "../../../../ui/skeleton";

export const Route = createFileRoute("/_app/settings/media-servers/$serverId")({
  staticData: { title: loaderTitle },
  loader: async ({ context: { queryClient }, params: { serverId } }) => {
    const [server] = await Promise.all([
      ensureQuery(queryClient, getGetMediaServerSuspenseQueryOptions(serverId)),
      ensureQuery(queryClient, getListMediaServerKindsSuspenseQueryOptions()),
    ]);
    return {
      title: server.name,
      trail: [
        { title: "Settings", pathname: "/settings" },
        { title: "Media Servers", pathname: "/settings/media-servers" },
      ],
    };
  },
  pendingComponent: () => <Skeleton label="Loading the Media Server" />,
  component: function MediaServer() {
    return <ServerLayout serverId={Route.useParams().serverId} />;
  },
});
