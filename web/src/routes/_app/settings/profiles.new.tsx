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
import { newProfileDraft, profileBodyOf } from "../../../targets/profile-draft";
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
            const made = await createProfile(profileBodyOf(draft));
            toast(`Created ${made.name}.`);
            await navigate({ to: "/settings/$section", params: { section: "profiles" } });
          },
        }}
      />
    </Page>
  );
}
