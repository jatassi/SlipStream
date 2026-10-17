import { createFileRoute } from "@tanstack/react-router";

import {
  getGetTargetSuspenseQueryOptions,
  getListQualitiesSuspenseQueryOptions,
  useGetTargetSuspense,
  useListQualitiesSuspense,
} from "../../../api/gen/query/targets/targets";
import { ensureQuery } from "../../../api/query-client";
import { Page } from "../../../shell/page";
import { loaderTitle } from "../../../shell/titles";
import { SavedTargetEditor } from "../../../targets/target-editor";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/settings/targets/$targetId")({
  staticData: { title: loaderTitle },
  loader: async ({ context, params: { targetId } }) => {
    const [target] = await Promise.all([
      ensureQuery(context.queryClient, getGetTargetSuspenseQueryOptions(targetId)),
      ensureQuery(context.queryClient, getListQualitiesSuspenseQueryOptions()),
    ]);
    return {
      title: target.name,
      trail: [
        { title: "Settings", pathname: "/settings" },
        { title: "Targets", pathname: "/settings/targets" },
      ],
    };
  },
  pendingComponent: () => <Skeleton label="Loading the Target" />,
  component: TargetPage,
});

function TargetPage() {
  const { targetId } = Route.useParams();
  const target = useGetTargetSuspense(targetId).data;
  const qualities = useListQualitiesSuspense().data;
  return (
    <Page>
      <SavedTargetEditor key={target.id} target={target} qualities={qualities} />
    </Page>
  );
}
