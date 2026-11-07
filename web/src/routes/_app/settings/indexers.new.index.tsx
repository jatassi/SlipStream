import { createFileRoute } from "@tanstack/react-router";

import {
  getGetIndexerCatalogSuspenseQueryOptions,
  useGetIndexerCatalogSuspense,
} from "../../../api/gen/query/indexers/indexers";
import { ensureQuery } from "../../../api/query-client";
import { SourceStep } from "../../../indexers/add/source-step";
import { addTrail } from "../../../indexers/add/trail";
import { Page } from "../../../shell/page";
import { loaderTitle } from "../../../shell/titles";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/settings/indexers/new/")({
  staticData: { title: loaderTitle },
  loader: async ({ context }) => {
    await ensureQuery(context.queryClient, getGetIndexerCatalogSuspenseQueryOptions());
    return { title: "Add Indexer", trail: addTrail };
  },
  pendingComponent: () => <Skeleton label="Loading the Indexer catalog" />,
  component: AddIndexerPage,
});

function AddIndexerPage() {
  const catalog = useGetIndexerCatalogSuspense().data;
  return (
    <Page>
      <SourceStep catalog={catalog} />
    </Page>
  );
}
