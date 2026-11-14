import { createFileRoute } from "@tanstack/react-router";

import {
  getListMediaServerLibrariesSuspenseQueryOptions,
  useGetMediaServerSuspense,
} from "../../../../api/gen/query/mediaservers/mediaservers";
import { ensureQuery } from "../../../../api/query-client";
import { MappingsTab } from "../../../../media-servers/page/mappings-tab";
import { serverSectionOf } from "../../../../media-servers/page/sections";
import { Skeleton } from "../../../../ui/skeleton";

const section = serverSectionOf("mappings");

export const Route = createFileRoute("/_app/settings/media-servers/$serverId/mappings")({
  staticData: { title: section.label },
  loader: ({ context: { queryClient }, params: { serverId } }) =>
    ensureQuery(queryClient, getListMediaServerLibrariesSuspenseQueryOptions(serverId)),
  pendingComponent: () => <Skeleton label="Loading the Path mappings" />,
  component: function Mappings() {
    return <MappingsTab server={useGetMediaServerSuspense(Route.useParams().serverId).data} />;
  },
});
