import { createFileRoute } from "@tanstack/react-router";

import { getListNodesSuspenseQueryOptions } from "../../../api/gen/query/nodes/nodes";
import { getGetRemoteAccessSuspenseQueryOptions } from "../../../api/gen/query/reachability/reachability";
import { ensureQuery, freshQuery } from "../../../api/query-client";
import { RemoteAccessPage } from "../../../remote-access/remote-access-page";
import { PagePending } from "../../../shell/page";

export const Route = createFileRoute("/_app/nodes/remote-access")({
  staticData: { title: "Remote access" },
  loader: async ({ context }) => {
    await Promise.all([
      freshQuery(context.queryClient, getGetRemoteAccessSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListNodesSuspenseQueryOptions()),
    ]);
  },
  pendingComponent: () => <PagePending label="Loading Remote access" shape="rows" lines={2} />,
  component: RemoteAccessPage,
});
