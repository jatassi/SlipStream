import { createFileRoute } from "@tanstack/react-router";

import {
  getListMediaServerCredentialHoldersSuspenseQueryOptions,
  useGetMediaServerSuspense,
} from "../../../../api/gen/query/mediaservers/mediaservers";
import { ensureQuery } from "../../../../api/query-client";
import { DetailsTab } from "../../../../media-servers/page/details-tab";
import { loaderTitle } from "../../../../shell/titles";
import { Skeleton } from "../../../../ui/skeleton";

export const Route = createFileRoute("/_app/settings/media-servers/$serverId/")({
  staticData: { title: loaderTitle },
  loader: ({ context: { queryClient }, params: { serverId } }) =>
    ensureQuery(queryClient, getListMediaServerCredentialHoldersSuspenseQueryOptions(serverId)),
  pendingComponent: () => <Skeleton label="Loading Details" />,
  component: function Details() {
    return <DetailsTab server={useGetMediaServerSuspense(Route.useParams().serverId).data} />;
  },
});
