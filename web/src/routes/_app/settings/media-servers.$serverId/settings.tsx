import { createFileRoute } from "@tanstack/react-router";

import { useGetMediaServerSuspense } from "../../../../api/gen/query/mediaservers/mediaservers";
import { getListNodesSuspenseQueryOptions } from "../../../../api/gen/query/nodes/nodes";
import { ensureQuery } from "../../../../api/query-client";
import { SettingsTab } from "../../../../media-servers/page/settings-tab";
import { serverSectionOf } from "../../../../media-servers/page/sections";
import { Skeleton } from "../../../../ui/skeleton";

const section = serverSectionOf("settings");

export const Route = createFileRoute("/_app/settings/media-servers/$serverId/settings")({
  staticData: { title: section.label },
  loader: ({ context: { queryClient } }) =>
    ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
  pendingComponent: () => <Skeleton label="Loading Settings" />,
  component: function Settings() {
    return <SettingsTab server={useGetMediaServerSuspense(Route.useParams().serverId).data} />;
  },
});
