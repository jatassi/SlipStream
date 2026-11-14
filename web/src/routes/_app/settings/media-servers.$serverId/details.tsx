import { createFileRoute } from "@tanstack/react-router";

import {
  getListMediaServerCredentialHoldersSuspenseQueryOptions,
  useGetMediaServerSuspense,
} from "../../../../api/gen/query/mediaservers/mediaservers";
import { ensureQuery } from "../../../../api/query-client";
import { DetailsTab } from "../../../../media-servers/page/details-tab";
import { serverSectionOf } from "../../../../media-servers/page/sections";
import { Skeleton } from "../../../../ui/skeleton";

const section = serverSectionOf("details");

export const Route = createFileRoute("/_app/settings/media-servers/$serverId/details")({
  staticData: { title: section.label },
  loader: ({ context: { queryClient }, params: { serverId } }) =>
    ensureQuery(queryClient, getListMediaServerCredentialHoldersSuspenseQueryOptions(serverId)),
  pendingComponent: () => <Skeleton label="Loading Details" />,
  component: function Details() {
    return <DetailsTab server={useGetMediaServerSuspense(Route.useParams().serverId).data} />;
  },
});
