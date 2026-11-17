import { Sorting01Icon } from "@hugeicons/core-free-icons";
import { Link } from "@tanstack/react-router";

import type { WorksContentType } from "../api/gen/model";
import { LibraryViews } from "../library/library-views";
import { Button } from "../ui/button";
import { Menu } from "../ui/menu";
import { SegmentedLinks, segmentedLink } from "../ui/segmented-links";
import { type Reason, type Sort, reasonLabels, reasons, sortLabels, sorts } from "./wanted-words";

const exact = { exact: true, includeSearch: false };

/** All, then each enabled Content Type's Wanted, as the Library's links are. */
function TypeLinks({ types }: { types: readonly WorksContentType[] }) {
  return (
    <SegmentedLinks label="Content Types">
      <Link to="/library/wanted" activeOptions={exact} className={segmentedLink}>
        All
      </Link>
      {types.map((type) => (
        <Link
          key={type.id}
          to="/library/wanted/$contentType"
          params={{ contentType: type.id }}
          activeOptions={exact}
          className={segmentedLink}
        >
          {type.name}
        </Link>
      ))}
    </SegmentedLinks>
  );
}

/** The Library views, then the Content Type links while more than one type is enabled. */
export function WantedSwitches({
  types,
  contentType,
}: {
  types: readonly WorksContentType[];
  contentType: string | undefined;
}) {
  return (
    <>
      <LibraryViews contentType={contentType} />
      {types.length > 1 && <TypeLinks types={types} />}
    </>
  );
}

const chipLook =
  "inline-flex h-6 shrink-0 items-center rounded-chip px-2 text-chip whitespace-nowrap transition-colors duration-(--duration-control) hover:text-ink-1 aria-pressed:bg-selected aria-pressed:text-ink-1 not-aria-pressed:bg-fill not-aria-pressed:text-ink-2";

/** A toggle chip per reason, in the order a row picks its first; one stays pressed. */
export function ReasonChips({
  chosen,
  onChange,
}: {
  chosen: readonly Reason[];
  onChange: (next: Reason[]) => void;
}) {
  return (
    <fieldset
      aria-label="Reasons"
      className="flex min-w-0 gap-1.5 overflow-x-auto [scrollbar-width:none]"
    >
      {reasons.map((reason) => {
        const pressed = chosen.includes(reason);
        return (
          <button
            key={reason}
            type="button"
            aria-pressed={pressed}
            className={chipLook}
            onClick={() => {
              const next = pressed
                ? chosen.filter((each) => each !== reason)
                : reasons.filter((each) => each === reason || chosen.includes(each));
              if (next.length > 0) {
                onChange(next);
              }
            }}
          >
            {reasonLabels[reason]}
          </button>
        );
      })}
    </fieldset>
  );
}

export function SortMenu({ sort, onSort }: { sort: Sort; onSort: (sort: Sort) => void }) {
  return (
    <Menu
      label="Sort"
      trigger={<Button icon={Sorting01Icon}>{`Sort: ${sortLabels[sort]}`}</Button>}
      items={sorts.map((each) => ({
        kind: "toggle" as const,
        label: sortLabels[each],
        checked: each === sort,
        onCheckedChange: () => {
          onSort(each);
        },
      }))}
    />
  );
}
