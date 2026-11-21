import { createFileRoute, useNavigate } from "@tanstack/react-router";

import {
  getListContentTypesSuspenseQueryOptions,
  useListContentTypesSuspense,
} from "../../../api/gen/query/works/works";
import { ensureQuery } from "../../../api/query-client";
import { PagePending } from "../../../shell/page";
import { filterOf, searchOf, wantedSearch } from "../../../wanted/routed";
import { WantedPage } from "../../../wanted/wanted-page";
import { loadWanted } from "../../../wanted/wanted-query";
import { enabledTypeOf, enabledTypes } from "../../../works/fields";

// Un-nested from /library/wanted, so a phone's back link goes to the Library.
export const Route = createFileRoute("/_app/library/wanted_/$contentType")({
  staticData: { title: "Wanted" },
  validateSearch: wantedSearch,
  loaderDeps: ({ search }) => search,
  loader: async ({ context: { queryClient }, params: { contentType }, deps }) => {
    const { contentTypes } = await ensureQuery(
      queryClient,
      getListContentTypesSuspenseQueryOptions(),
    );
    enabledTypeOf(contentTypes, contentType);
    await loadWanted(queryClient, filterOf(deps, contentType));
  },
  pendingComponent: () => <PagePending label="Loading Wanted" shape="table" lines={2} />,
  component: ContentTypeWanted,
});

function ContentTypeWanted() {
  const navigate = useNavigate();
  const { contentType } = Route.useParams();
  const filter = filterOf(Route.useSearch(), contentType);
  const { contentTypes } = useListContentTypesSuspense().data;
  return (
    <WantedPage
      types={enabledTypes(contentTypes)}
      filter={filter}
      onFilter={(next) => {
        void navigate({
          to: "/library/wanted/$contentType",
          params: { contentType },
          search: searchOf({ ...filter, ...next }),
          replace: true,
        });
      }}
    />
  );
}
