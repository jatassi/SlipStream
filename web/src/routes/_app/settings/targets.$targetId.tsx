import { createFileRoute } from "@tanstack/react-router";

import {
  getListCustomFormatsSuspenseQueryOptions,
  useListCustomFormatsSuspense,
} from "../../../api/gen/query/formats/formats";
import {
  getGetTargetSuspenseQueryOptions,
  getListQualitiesSuspenseQueryOptions,
  useGetTargetSuspense,
  useListQualitiesSuspense,
} from "../../../api/gen/query/targets/targets";
import { ensureQuery, freshQuery } from "../../../api/query-client";
import { Page } from "../../../shell/page";
import { loaderTitle } from "../../../shell/titles";
import { SavedTargetEditor } from "../../../targets/target-editor";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/settings/targets/$targetId")({
  staticData: { title: loaderTitle },
  loader: async ({ context, params: { targetId } }) => {
    const [target] = await Promise.all([
      freshQuery(context.queryClient, getGetTargetSuspenseQueryOptions(targetId)),
      ensureQuery(context.queryClient, getListQualitiesSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListCustomFormatsSuspenseQueryOptions()),
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
  const { formats } = useListCustomFormatsSuspense().data;
  const navigate = Route.useNavigate();
  return (
    <Page>
      <SavedTargetEditor
        key={target.id}
        target={target}
        qualities={qualities}
        formats={formats}
        onSaved={() => navigate({ to: "/settings/$section", params: { section: "targets" } })}
      />
    </Page>
  );
}
