import { createFileRoute } from "@tanstack/react-router";

import { useGetMediaServerSuspense } from "../../../../api/gen/query/mediaservers/mediaservers";
import { loadChanges } from "../../../../media-servers/page/changes-query";
import { ChangesTab } from "../../../../media-servers/page/changes-tab";
import { serverSectionOf } from "../../../../media-servers/page/sections";
import { Skeleton } from "../../../../ui/skeleton";

const section = serverSectionOf("changes");

export const Route = createFileRoute("/_app/settings/media-servers/$serverId/changes")({
  staticData: { title: section.label },
  loader: ({ context: { queryClient }, params: { serverId } }) =>
    loadChanges(queryClient, serverId),
  pendingComponent: () => <Skeleton label="Loading the changes" />,
  component: function Changes() {
    return <ChangesTab server={useGetMediaServerSuspense(Route.useParams().serverId).data} />;
  },
});
