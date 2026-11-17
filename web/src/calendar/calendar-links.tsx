import { Link } from "@tanstack/react-router";
import type { ReactNode } from "react";

import type { WorksContentType } from "../api/gen/model";
import { LibraryViews } from "../library/library-views";
import { SegmentedLinks, segmentedLink } from "../ui/segmented-links";
import { type CalendarSearch, type CalendarView, calendarViews } from "./routed";

type Active = { exact: boolean; includeSearch: boolean };

function CalendarLink({
  contentType,
  search,
  active,
  children,
}: {
  contentType: string | undefined;
  search: CalendarSearch;
  active: Active;
  children: ReactNode;
}) {
  return contentType === undefined ? (
    <Link to="/library/calendar" search={search} activeOptions={active} className={segmentedLink}>
      {children}
    </Link>
  ) : (
    <Link
      to="/library/calendar/$contentType"
      params={{ contentType }}
      search={search}
      activeOptions={active}
      className={segmentedLink}
    >
      {children}
    </Link>
  );
}

const typeActive = { exact: true, includeSearch: false };

/** All, then each enabled Content Type's Calendar, as the Library's links are, keeping the view and place. */
function CalendarTypeLinks({
  types,
  search,
}: {
  types: readonly WorksContentType[];
  search: CalendarSearch;
}) {
  const { kinds: _kinds, ...kept } = search;
  return (
    <SegmentedLinks label="Content Types">
      <CalendarLink contentType={undefined} search={kept} active={typeActive}>
        All
      </CalendarLink>
      {types.map((type) => (
        <CalendarLink key={type.id} contentType={type.id} search={kept} active={typeActive}>
          {type.name}
        </CalendarLink>
      ))}
    </SegmentedLinks>
  );
}

/** The Library views, then the Content Type links while more than one type is enabled. */
export function CalendarSwitches({
  types,
  contentType,
  search,
}: {
  types: readonly WorksContentType[];
  contentType: string | undefined;
  search: CalendarSearch;
}) {
  return (
    <>
      <LibraryViews contentType={contentType} />
      {types.length > 1 && <CalendarTypeLinks types={types} search={search} />}
    </>
  );
}

const viewLabels: { readonly [V in CalendarView]: string } = { month: "Month", list: "List" };
const viewActive = { exact: false, includeSearch: true };

/** Month and List, each its own address: Month keeps the month and List its first day. */
export function ViewLinks({
  contentType,
  search,
}: {
  contentType: string | undefined;
  search: CalendarSearch;
}) {
  const { month, from, ...rest } = search;
  const searchOf = (view: CalendarView): CalendarSearch =>
    view === "month" ? { ...rest, month, view } : { ...rest, from, view };
  return (
    <SegmentedLinks label="View">
      {calendarViews.map((view) => (
        <CalendarLink
          key={view}
          contentType={contentType}
          search={searchOf(view)}
          active={viewActive}
        >
          {viewLabels[view]}
        </CalendarLink>
      ))}
    </SegmentedLinks>
  );
}
