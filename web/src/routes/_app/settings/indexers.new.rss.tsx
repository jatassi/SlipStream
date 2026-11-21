import { createFileRoute } from "@tanstack/react-router";

import {
  getListNodesSuspenseQueryOptions,
  useListNodesSuspense,
} from "../../../api/gen/query/nodes/nodes";
import {
  getListContentTypesSuspenseQueryOptions,
  useListContentTypesSuspense,
} from "../../../api/gen/query/works/works";
import { ensureQuery } from "../../../api/query-client";
import { AddIndexerForm } from "../../../indexers/add/add-form";
import { addStepTrail } from "../../../indexers/add/trail";
import { Page, PagePending } from "../../../shell/page";
import { loaderTitle } from "../../../shell/titles";

export const Route = createFileRoute("/_app/settings/indexers/new/rss")({
  staticData: { title: loaderTitle },
  loader: async ({ context }) => {
    await Promise.all([
      ensureQuery(context.queryClient, getListNodesSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListContentTypesSuspenseQueryOptions()),
    ]);
    return { title: "Add an RSS feed", trail: addStepTrail };
  },
  pendingComponent: () => <PagePending label="Loading the form" shape="form" />,
  component: AddRssPage,
});

function AddRssPage() {
  const { nodes } = useListNodesSuspense().data;
  const { contentTypes } = useListContentTypesSuspense().data;
  return (
    <Page>
      <AddIndexerForm source={{ kind: "rss", contentTypes }} nodes={nodes} />
    </Page>
  );
}
