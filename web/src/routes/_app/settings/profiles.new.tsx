import { createFileRoute } from "@tanstack/react-router";

import {
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
import { NewProfileEditor } from "../../../targets/profile-editor";
import { Skeleton } from "../../../ui/skeleton";

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
  const type = contentTypes.find((candidate) => candidate.enabled) ?? contentTypes[0];
  return (
    <Page>
      <NewProfileEditor contentType={type?.id ?? ""} />
    </Page>
  );
}
