import { createFileRoute, useNavigate } from "@tanstack/react-router";

import { getListContentTypesSuspenseQueryOptions } from "../../../api/gen/query/works/works";
import { ensureQuery } from "../../../api/query-client";
import { HistoryPage } from "../../../history/history-page";
import { loadHistory } from "../../../history/history-query";
import { type HistoryFilter, historySearch, paramsOf } from "../../../history/routed";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/activity/history")({
  staticData: { title: "History", view: true },
  validateSearch: historySearch,
  loaderDeps: ({ search: { type, groups, kinds, since, work } }) => ({
    type,
    groups,
    kinds,
    since,
    work,
  }),
  loader: ({ context: { queryClient }, deps }) => {
    const now = new Date();
    return Promise.all([
      loadHistory(queryClient, paramsOf(deps, now), now),
      ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
    ]);
  },
  pendingComponent: () => <Skeleton label="Loading History" />,
  component: ActivityHistory,
});

function ActivityHistory() {
  const navigate = useNavigate();
  const { type, groups, kinds, since, work } = Route.useSearch();
  return (
    <HistoryPage
      filter={{ type, groups, kinds, since, work }}
      onFilter={(change: Partial<HistoryFilter>) => {
        void navigate({
          to: "/activity/history",
          search: (prev) => ({ ...prev, ...change }),
          replace: true,
        });
      }}
    />
  );
}
