import { createFileRoute, useNavigate } from "@tanstack/react-router";

import {
  getListCustomFormatsSuspenseQueryOptions,
  useListCustomFormatsSuspense,
} from "../../../api/gen/query/formats/formats";
import {
  createTarget,
  getListQualitiesSuspenseQueryOptions,
  useListQualitiesSuspense,
} from "../../../api/gen/query/targets/targets";
import { ensureQuery } from "../../../api/query-client";
import { Page, PagePending } from "../../../shell/page";
import { bodyOf, newDraft } from "../../../targets/draft";
import { TargetEditor } from "../../../targets/target-editor";
import { toast } from "../../../ui/toast";
import { loaderTitle } from "../../../shell/titles";

export const Route = createFileRoute("/_app/settings/targets/new")({
  staticData: { title: loaderTitle },
  loader: async ({ context }) => {
    await Promise.all([
      ensureQuery(context.queryClient, getListQualitiesSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListCustomFormatsSuspenseQueryOptions()),
    ]);
    return {
      title: "New Target",
      trail: [
        { title: "Settings", pathname: "/settings" },
        { title: "Targets", pathname: "/settings/targets" },
      ],
    };
  },
  pendingComponent: () => <PagePending label="Loading the Target editor" shape="form" />,
  component: NewTargetPage,
});

function NewTargetPage() {
  const qualities = useListQualitiesSuspense().data;
  const { formats } = useListCustomFormatsSuspense().data;
  const navigate = useNavigate();
  return (
    <Page>
      <TargetEditor
        initial={newDraft(qualities, formats)}
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
