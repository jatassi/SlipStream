import { createFileRoute } from "@tanstack/react-router";

import {
  getGetSystemHealthSuspenseQueryOptions,
  useGetSystemHealthSuspense,
} from "../../api/gen/query/health/health";
import { ensureQuery } from "../../api/query-client";
import { HealthPage } from "../../health/health-page";
import { Skeleton } from "../../ui/skeleton";

export const Route = createFileRoute("/_app/health")({
  staticData: { title: "Health" },
  loader: ({ context }) =>
    ensureQuery(context.queryClient, getGetSystemHealthSuspenseQueryOptions()),
  pendingComponent: () => <Skeleton label="Loading Health" />,
  component: Health,
});

function Health() {
  return <HealthPage report={useGetSystemHealthSuspense().data} />;
}
