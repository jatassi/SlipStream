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
import { enabledTypes } from "../../../works/fields";

export const Route = createFileRoute("/_app/library/wanted")({
  staticData: { title: "Wanted" },
  validateSearch: wantedSearch,
  loaderDeps: ({ search }) => search,
  loader: async ({ context: { queryClient }, deps }) => {
    await Promise.all([
      ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
      loadWanted(queryClient, filterOf(deps, undefined)),
    ]);
  },
  pendingComponent: () => <PagePending label="Loading Wanted" shape="table" lines={2} />,
  component: Wanted,
});

function Wanted() {
  const navigate = useNavigate();
  const filter = filterOf(Route.useSearch(), undefined);
  const { contentTypes } = useListContentTypesSuspense().data;
  return (
    <WantedPage
      types={enabledTypes(contentTypes)}
      filter={filter}
      onFilter={(next) => {
        void navigate({
          to: "/library/wanted",
          search: searchOf({ ...filter, ...next }),
          replace: true,
        });
      }}
    />
  );
}
