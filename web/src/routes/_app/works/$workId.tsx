import { createFileRoute } from "@tanstack/react-router";

import { getGetWorkCellsSuspenseQueryOptions } from "../../../api/gen/query/files/files";
import {
  getGetWorkSuspenseQueryOptions,
  getListContentTypesSuspenseQueryOptions,
  useGetWorkSuspense,
} from "../../../api/gen/query/works/works";
import {
  getGetWorkWantsSuspenseQueryOptions,
  getListCutsSuspenseQueryOptions,
  getListProfilesSuspenseQueryOptions,
} from "../../../api/gen/query/targets/targets";
import { ensureQuery } from "../../../api/query-client";
import { loadWorkGrabs } from "../../../grabs/waiting";
import { useAmbient } from "../../../shell/ambient/use-ambient";
import { loaderTitle } from "../../../shell/titles";
import { Skeleton } from "../../../ui/skeleton";
import { libraryTrail } from "../../../works/fields";
// oxlint-disable-next-line import/max-dependencies -- the page loads every section's queries, the Profile and Editions sheet's included
import { WorkBody } from "./-work-body";

export const Route = createFileRoute("/_app/works/$workId")({
  staticData: { title: loaderTitle },
  loader: async ({ context, params: { workId } }) => {
    const [{ contentTypes }, work] = await Promise.all([
      ensureQuery(context.queryClient, getListContentTypesSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getGetWorkSuspenseQueryOptions(workId)),
      ensureQuery(context.queryClient, getGetWorkCellsSuspenseQueryOptions(workId)),
      loadWorkGrabs(context.queryClient, workId),
      ensureQuery(context.queryClient, getGetWorkWantsSuspenseQueryOptions(workId)),
      ensureQuery(context.queryClient, getListProfilesSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListCutsSuspenseQueryOptions()),
    ]);
    return {
      title: work.label,
      trail: libraryTrail(contentTypes, work.contentType),
    };
  },
  pendingComponent: () => <Skeleton label="Loading the Work" />,
  component: WorkPage,
});

function WorkPage() {
  const { workId } = Route.useParams();
  const work = useGetWorkSuspense(workId).data;
  useAmbient(neutral);
  return <WorkBody workId={workId} work={work} />;
}

const neutral = { kind: "neutral" } as const;
