import { createFileRoute } from "@tanstack/react-router";

import {
  getListNodesSuspenseQueryOptions,
  useListNodesSuspense,
} from "../../../api/gen/query/nodes/nodes";
import { ensureQuery } from "../../../api/query-client";
import { AddIndexerForm } from "../../../indexers/add/add-form";
import { addStepTrail } from "../../../indexers/add/trail";
import { Page } from "../../../shell/page";
import { loaderTitle } from "../../../shell/titles";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/settings/indexers/new/torznab")({
  staticData: { title: loaderTitle },
  loader: async ({ context }) => {
    await ensureQuery(context.queryClient, getListNodesSuspenseQueryOptions());
    return { title: "Add a Torznab feed", trail: addStepTrail };
  },
  pendingComponent: () => <Skeleton label="Loading the form" />,
  component: AddTorznabPage,
});

function AddTorznabPage() {
  const { nodes } = useListNodesSuspense().data;
  return (
    <Page>
      <AddIndexerForm source={{ kind: "torznab" }} nodes={nodes} />
    </Page>
  );
}
