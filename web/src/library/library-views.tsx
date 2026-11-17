import { Link } from "@tanstack/react-router";

import { SegmentedLinks, segmentedLink } from "../ui/segmented-links";

const views = [
  { label: "Library", all: "/library", one: "/library/$contentType" },
  { label: "Wanted", all: "/library/wanted", one: "/library/wanted/$contentType" },
  { label: "Calendar", all: "/library/calendar", one: "/library/calendar/$contentType" },
] as const;

const exact = { exact: true, includeSearch: false };

/**
 * The switch between the Library, Wanted and the Calendar, of `contentType`
 * or of every Content Type, at the head of each of the three.
 */
export function LibraryViews({ contentType }: { contentType: string | undefined }) {
  return (
    <SegmentedLinks label="Library views">
      {views.map(({ label, all, one }) =>
        contentType === undefined ? (
          <Link key={label} to={all} activeOptions={exact} className={segmentedLink}>
            {label}
          </Link>
        ) : (
          <Link
            key={label}
            to={one}
            params={{ contentType }}
            activeOptions={exact}
            className={segmentedLink}
          >
            {label}
          </Link>
        ),
      )}
    </SegmentedLinks>
  );
}
