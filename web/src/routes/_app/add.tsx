import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import {
  getListContentTypesSuspenseQueryOptions,
  getSearchCatalogSuspenseQueryOptions,
} from "../../api/gen/query/works/works";
import { ensureQuery } from "../../api/query-client";
import { loaderTitle } from "../../shell/titles";
import { Page } from "../../shell/page";
import { Skeleton } from "../../ui/skeleton";
import { addLabel, enabledTypeOf } from "../../works/fields";
import { Candidates } from "./-candidates";
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
      await ensureQuery(
        context.queryClient,
        getSearchCatalogSuspenseQueryOptions(type.id, { query }),
      );
    }
    return { title: addLabel(type), type };
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
