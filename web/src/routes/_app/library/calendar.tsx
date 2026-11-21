import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router";

import {
  getListContentTypesSuspenseQueryOptions,
  useListContentTypesSuspense,
} from "../../../api/gen/query/works/works";
import { ensureQuery } from "../../../api/query-client";
import { CalendarPage } from "../../../calendar/calendar-page";
import { loadCalendar } from "../../../calendar/calendar-query";
import { calendarSearch, defaultView } from "../../../calendar/routed";
import { PagePending } from "../../../shell/page";
import { enabledTypes } from "../../../works/fields";

export const Route = createFileRoute("/_app/library/calendar")({
  staticData: { title: "Calendar" },
  validateSearch: calendarSearch,
  beforeLoad: ({ search }) => {
    if (search.view === undefined) {
      // oxlint-disable-next-line typescript/only-throw-error -- the router follows a thrown redirect()
      throw redirect({
        to: "/library/calendar",
        search: { ...search, view: defaultView() },
        replace: true,
      });
    }
  },
  loaderDeps: ({ search }) => search,
  loader: async ({ context: { queryClient }, deps }) => {
    await Promise.all([
      ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
      loadCalendar(queryClient, deps, undefined),
    ]);
  },
  pendingComponent: () => <PagePending label="Loading the Calendar" shape="table" lines={2} />,
  component: Calendar,
});

function Calendar() {
  const navigate = useNavigate();
  const search = Route.useSearch();
  const { contentTypes } = useListContentTypesSuspense().data;
  return (
    <CalendarPage
      types={enabledTypes(contentTypes)}
      contentType={undefined}
      search={search}
      onSearch={(next) => {
        void navigate({ to: "/library/calendar", search: next, replace: true });
      }}
    />
  );
}
