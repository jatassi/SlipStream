import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { getListContentTypesSuspenseQueryOptions } from "../../api/gen/query/works/works";
import { ensureQuery } from "../../api/query-client";
import { Page } from "../../shell/page";
import { loaderTitle } from "../../shell/titles";
import { Skeleton } from "../../ui/skeleton";
import { loadCandidates } from "../../works/adding";
import { addLabel, enabledTypeOf, libraryTrail } from "../../works/fields";
import { Candidates } from "./-candidates";
// oxlint-disable-next-line import/max-dependencies -- the page loads the catalog and the add sheet's Profiles
import { CatalogSearch, searchForm } from "./-catalog-search";

export const Route = createFileRoute("/_app/add")({
  staticData: { title: loaderTitle },
  validateSearch: z.object({
    contentType: z.string().catch(""),
    query: searchForm.shape.query.optional().catch(undefined),
  }),
  loaderDeps: ({ search: { contentType, query } }) => ({ contentType, query }),
  loader: async ({ context, deps: { contentType, query } }) => {
    const { contentTypes } = await ensureQuery(
      context.queryClient,
      getListContentTypesSuspenseQueryOptions(),
    );
    const type = enabledTypeOf(contentTypes, contentType);
    if (query !== undefined) {
      await loadCandidates(context.queryClient, type, query);
    }
    return { title: addLabel(type), trail: libraryTrail(contentTypes, type.id), type };
  },
  pendingComponent: () => <Skeleton label="Loading the catalog" />,
  component: AddPage,
});

function AddPage() {
  const { type } = Route.useLoaderData();
  const { query } = Route.useSearch();
  return (
    <Page>
      <CatalogSearch />
      {query === undefined ? null : <Candidates type={type} query={query} />}
    </Page>
  );
}
