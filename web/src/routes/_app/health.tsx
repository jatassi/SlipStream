import { createFileRoute } from "@tanstack/react-router";

import {
  getGetHealthSettingsSuspenseQueryOptions,
  getGetSystemHealthSuspenseQueryOptions,
  getListHealthVolumesSuspenseQueryOptions,
  useGetSystemHealthSuspense,
} from "../../api/gen/query/health/health";
import { getGetRootSettingsSuspenseQueryOptions } from "../../api/gen/query/roots/roots";
import { ensureQuery, freshQuery } from "../../api/query-client";
import { HealthPage } from "../../health/health-page";
import { Skeleton } from "../../ui/skeleton";

export const Route = createFileRoute("/_app/health")({
  staticData: { title: "Health" },
  loader: async ({ context: { queryClient } }) => {
    await Promise.all([
      ensureQuery(queryClient, getGetSystemHealthSuspenseQueryOptions()),
      ensureQuery(queryClient, getGetHealthSettingsSuspenseQueryOptions()),
      ensureQuery(queryClient, getListHealthVolumesSuspenseQueryOptions()),
      freshQuery(queryClient, getGetRootSettingsSuspenseQueryOptions()),
    ]);
  },
  pendingComponent: () => <Skeleton label="Loading Health" />,
  component: Health,
});

function Health() {
  return <HealthPage report={useGetSystemHealthSuspense().data} />;
}
