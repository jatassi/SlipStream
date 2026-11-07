import { createFileRoute } from "@tanstack/react-router";

import {
  getGetCatalogDefinitionSuspenseQueryOptions,
  useGetCatalogDefinitionSuspense,
} from "../../../api/gen/query/indexers/indexers";
import {
  getListNodesSuspenseQueryOptions,
  useListNodesSuspense,
} from "../../../api/gen/query/nodes/nodes";
import { ensureQuery, freshQuery } from "../../../api/query-client";
import { DefinitionStep } from "../../../indexers/add/definition-step";
import { addStepTrail } from "../../../indexers/add/trail";
import { Page } from "../../../shell/page";
import { loaderTitle } from "../../../shell/titles";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/settings/indexers/new/catalog/$definitionId")({
  staticData: { title: loaderTitle },
  loader: async ({ context: { queryClient }, params: { definitionId } }) => {
    const [definition] = await Promise.all([
      freshQuery(queryClient, getGetCatalogDefinitionSuspenseQueryOptions(definitionId)),
      ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
    ]);
    return { title: `Add ${definition.entry.name}`, trail: addStepTrail };
  },
  pendingComponent: () => <Skeleton label="Loading the definition" />,
  component: AddDefinitionPage,
});

function AddDefinitionPage() {
  const { definitionId } = Route.useParams();
  const definition = useGetCatalogDefinitionSuspense(definitionId).data;
  const { nodes } = useListNodesSuspense().data;
  return (
    <Page>
      <DefinitionStep key={definition.entry.version} definition={definition} nodes={nodes} />
    </Page>
  );
}
