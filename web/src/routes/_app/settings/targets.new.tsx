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
import { newTargetSearch, suggestedDraft, suggestedLine } from "../../../targets/suggested";
import { TargetEditor } from "../../../targets/target-editor";
import { toast } from "../../../ui/toast";
import { loaderTitle } from "../../../shell/titles";

export const Route = createFileRoute("/_app/settings/targets/new")({
  validateSearch: newTargetSearch,
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
  const { from } = Route.useSearch();
  const navigate = useNavigate();
  const base = newDraft(qualities, formats);
  return (
    <Page>
      {from && <p className="mb-4 text-pretty text-ink-1">{suggestedLine(from)}</p>}
      <TargetEditor
        initial={from ? suggestedDraft(base, from.suggestion, qualities.qualities) : base}
        save={{
          label: "Create Target",
          onSave: async (draft) => {
            const made = await createTarget(bodyOf(draft));
            toast(
              from
                ? `Created ${made.name}. Add it to a Profile so that Works get a file for it.`
                : `Created ${made.name}.`,
            );
            await navigate({ to: "/settings/$section", params: { section: "targets" } });
          },
        }}
      />
    </Page>
  );
}
