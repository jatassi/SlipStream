import { createFileRoute } from "@tanstack/react-router";

import {
  getGetProfileSuspenseQueryOptions,
  getListCutsSuspenseQueryOptions,
  getListTargetsSuspenseQueryOptions,
  useGetProfileSuspense,
} from "../../../api/gen/query/targets/targets";
import { getListContentTypesSuspenseQueryOptions } from "../../../api/gen/query/works/works";
import { ensureQuery } from "../../../api/query-client";
import { Page } from "../../../shell/page";
import { loaderTitle } from "../../../shell/titles";
import { profileDraftOf } from "../../../targets/profile-draft";
import { ProfileEditor } from "../../../targets/profile-editor";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/settings/profiles/$profileId")({
  staticData: { title: loaderTitle },
  loader: async ({ context, params: { profileId } }) => {
    const [profile] = await Promise.all([
      ensureQuery(context.queryClient, getGetProfileSuspenseQueryOptions(profileId)),
      ensureQuery(context.queryClient, getListContentTypesSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListTargetsSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListCutsSuspenseQueryOptions()),
    ]);
    return {
      title: profile.name,
      trail: [
        { title: "Settings", pathname: "/settings" },
        { title: "Profiles", pathname: "/settings/profiles" },
      ],
    };
  },
  pendingComponent: () => <Skeleton label="Loading the Profile" />,
  component: ProfilePage,
});

// Saving an existing Profile goes through the Impact preview, which a later
// unit wires in as the editor's save.
function ProfilePage() {
  const { profileId } = Route.useParams();
  const profile = useGetProfileSuspense(profileId).data;
  return (
    <Page>
      <ProfileEditor key={profile.id} initial={profileDraftOf(profile)} />
    </Page>
  );
}
