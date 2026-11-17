import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router";

import {
  getListContentTypesSuspenseQueryOptions,
  useListContentTypesSuspense,
} from "../../../api/gen/query/works/works";
import { ensureQuery } from "../../../api/query-client";
import { CalendarPage } from "../../../calendar/calendar-page";
import { loadCalendar } from "../../../calendar/calendar-query";
import { calendarSearch, defaultView } from "../../../calendar/routed";
import { Skeleton } from "../../../ui/skeleton";
import { enabledTypeOf, enabledTypes } from "../../../works/fields";

// Un-nested from /library/calendar, so a phone's back link goes to the Library.
export const Route = createFileRoute("/_app/library/calendar_/$contentType")({
  staticData: { title: "Calendar" },
  validateSearch: calendarSearch,
  beforeLoad: ({ search, params: { contentType } }) => {
    if (search.view === undefined) {
      // oxlint-disable-next-line typescript/only-throw-error -- the router follows a thrown redirect()
      throw redirect({
        to: "/library/calendar/$contentType",
        params: { contentType },
        search: { ...search, view: defaultView() },
        replace: true,
      });
    }
  },
  loaderDeps: ({ search }) => search,
  loader: async ({ context: { queryClient }, params: { contentType }, deps }) => {
    const { contentTypes } = await ensureQuery(
      queryClient,
      getListContentTypesSuspenseQueryOptions(),
    );
    enabledTypeOf(contentTypes, contentType);
    await loadCalendar(queryClient, deps, contentType);
  },
  pendingComponent: () => <Skeleton label="Loading the Calendar" />,
  component: ContentTypeCalendar,
});

function ContentTypeCalendar() {
  const navigate = useNavigate();
  const { contentType } = Route.useParams();
  const search = Route.useSearch();
  const { contentTypes } = useListContentTypesSuspense().data;
  return (
    <CalendarPage
      types={enabledTypes(contentTypes)}
      contentType={contentType}
      search={search}
      onSearch={(next) => {
        void navigate({
          to: "/library/calendar/$contentType",
          params: { contentType },
          search: next,
          replace: true,
        });
      }}
    />
  );
}
