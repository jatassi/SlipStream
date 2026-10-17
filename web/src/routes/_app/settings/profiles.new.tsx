import { createFileRoute, useNavigate } from "@tanstack/react-router";

import {
  createProfile,
  getListCutsSuspenseQueryOptions,
  getListTargetsSuspenseQueryOptions,
} from "../../../api/gen/query/targets/targets";
import {
  getListContentTypesSuspenseQueryOptions,
  useListContentTypesSuspense,
} from "../../../api/gen/query/works/works";
import { ensureQuery } from "../../../api/query-client";
import { Page } from "../../../shell/page";
import { loaderTitle } from "../../../shell/titles";
import { type ProfileDraft, newProfileDraft } from "../../../targets/profile-draft";
import { ProfileEditor } from "../../../targets/profile-editor";
import { Skeleton } from "../../../ui/skeleton";
import { toast } from "../../../ui/toast";

export const Route = createFileRoute("/_app/settings/profiles/new")({
  staticData: { title: loaderTitle },
  loader: async ({ context }) => {
    await Promise.all([
      ensureQuery(context.queryClient, getListContentTypesSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListTargetsSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListCutsSuspenseQueryOptions()),
    ]);
    return {
      title: "New Profile",
      trail: [
        { title: "Settings", pathname: "/settings" },
        { title: "Profiles", pathname: "/settings/profiles" },
      ],
    };
  },
  pendingComponent: () => <Skeleton label="Loading the Profile editor" />,
  component: NewProfilePage,
});

// Creating sends Any cut only: a chosen cut list is saved once the API
// stores one.
function creating(draft: ProfileDraft) {
  if (draft.editions.kind === "cuts") {
    throw new Error("Chosen cuts can't be saved yet. Choose Any cut to create the Profile.");
  }
  return createProfile({
    name: draft.name.trim(),
    contentType: draft.contentType,
    targets: draft.targets.map((targetId) => ({ targetId, sourcing: "download" })),
    editions: { kind: draft.editions.kind },
  });
}

function NewProfilePage() {
  const { contentTypes } = useListContentTypesSuspense().data;
  const navigate = useNavigate();
  const type = contentTypes.find((candidate) => candidate.enabled) ?? contentTypes[0];
  return (
    <Page>
      <ProfileEditor
        initial={newProfileDraft(type?.id ?? "")}
        save={{
          label: "Create Profile",
          onSave: async (draft) => {
            const made = await creating(draft);
            toast(`Created ${made.name}.`);
            await navigate({ to: "/settings/$section", params: { section: "profiles" } });
          },
        }}
      />
    </Page>
  );
}
