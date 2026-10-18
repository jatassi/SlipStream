import { createFileRoute, useNavigate } from "@tanstack/react-router";

import {
  createTarget,
  getListQualitiesSuspenseQueryOptions,
  useListQualitiesSuspense,
} from "../../../api/gen/query/targets/targets";
import { ensureQuery } from "../../../api/query-client";
import { Page } from "../../../shell/page";
import { bodyOf, newDraft } from "../../../targets/draft";
import { TargetEditor } from "../../../targets/target-editor";
import { toast } from "../../../ui/toast";
import { Skeleton } from "../../../ui/skeleton";
import { loaderTitle } from "../../../shell/titles";

export const Route = createFileRoute("/_app/settings/targets/new")({
  staticData: { title: loaderTitle },
  loader: async ({ context }) => {
    await ensureQuery(context.queryClient, getListQualitiesSuspenseQueryOptions());
    return {
      title: "New Target",
      trail: [
        { title: "Settings", pathname: "/settings" },
        { title: "Targets", pathname: "/settings/targets" },
      ],
    };
  },
  pendingComponent: () => <Skeleton label="Loading the Target editor" />,
  component: NewTargetPage,
});

function NewTargetPage() {
  const qualities = useListQualitiesSuspense().data;
  const navigate = useNavigate();
  return (
    <Page>
      <TargetEditor
        initial={newDraft(qualities)}
        save={{
          label: "Create Target",
          onSave: async (draft) => {
            const made = await createTarget(bodyOf(draft));
            toast(`Created ${made.name}.`);
            await navigate({ to: "/settings/$section", params: { section: "targets" } });
          },
        }}
      />
    </Page>
  );
}
