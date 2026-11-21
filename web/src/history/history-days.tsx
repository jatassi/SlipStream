import { useSuspenseInfiniteQuery } from "@tanstack/react-query";

import type { HistoryListEntry } from "../api/gen/model";
import { Button } from "../ui/button";
import { Empty } from "../ui/empty";
import { List } from "../ui/list";
import { NextOnScroll } from "../ui/next-on-scroll";
import { type Clock, useClock, useNow } from "../ui/clock";
import { useShell } from "../ui/shell-context";
import { whenWords } from "../ui/time-words";
import { historyPages } from "./history-query";
// oxlint-disable-next-line import/max-dependencies -- the list lays out its days, its empty states and its paging
import { HistoryRow } from "./history-row";
import { type HistoryFilter, filtered, paramsOf } from "./routed";

type Day = { readonly heading: string; readonly entries: HistoryListEntry[] };

/** entries, newest first, in runs of one day each, which its heading names. */
function daysOf(entries: readonly HistoryListEntry[], clock: Clock): Day[] {
  const days: Day[] = [];
  for (const entry of entries) {
    const heading = whenWords(entry.at, "heading", clock);
    const last = days.at(-1);
    if (last?.heading === heading) {
      last.entries.push(entry);
    } else {
      days.push({ heading, entries: [entry] });
    }
  }
  return days;
}

/** entries under their day's heading, the History page's whole list and the Work page's ten. */
export function DayLists({
  entries,
  poster = true,
}: {
  entries: readonly HistoryListEntry[];
  poster?: boolean;
}) {
  return daysOf(entries, useClock()).map(({ heading, entries: onDay }) => (
    <section key={heading} className="not-first:mt-5">
      <h3 className="mb-2 text-meta font-semibold text-ink-2">{heading}</h3>
      <List label={heading}>
        {onDay.map((entry) => (
          <HistoryRow key={entry.id} entry={entry} poster={poster} />
        ))}
      </List>
    </section>
  ));
}

const emptyWords =
  "History is empty. Grabs, Imports and Works added or deleted show here as they happen.";

/** Nothing matches: the filters hid every entry, or History holds none at all, as after a Clear. */
function FilteredEmpty({ now, onClear }: { now: Date; onClear: () => void }) {
  const { data } = useSuspenseInfiniteQuery(historyPages(paramsOf({}, now)));
  return data.pages[0]?.entries.length === 0 ? (
    <Empty look="sheet">{emptyWords}</Empty>
  ) : (
    <Empty look="sheet" action={<Button onClick={onClear}>Clear filters</Button>}>
      No entries match these filters.
    </Empty>
  );
}

/** Every loaded page's entries under their day's heading, ending in Load more on desktop and loading as a phone scrolls. */
export function HistoryDays({ filter, onClear }: { filter: HistoryFilter; onClear: () => void }) {
  const now = useNow();
  const phone = useShell() === "phone";
  const { data, hasNextPage, isFetchingNextPage, fetchNextPage } = useSuspenseInfiniteQuery(
    historyPages(paramsOf(filter, now)),
  );
  const entries = data.pages.flatMap((page) => page.entries);
  if (entries.length === 0) {
    return filtered(filter) ? (
      <FilteredEmpty now={now} onClear={onClear} />
    ) : (
      <Empty look="sheet">{emptyWords}</Empty>
    );
  }
  const more = () => {
    if (hasNextPage && !isFetchingNextPage) {
      void fetchNextPage();
    }
  };
  return (
    <>
      <DayLists entries={entries} />
      {hasNextPage &&
        (phone ? (
          <NextOnScroll onNext={more} />
        ) : (
          <div className="mt-4 flex justify-center">
            <Button busy={isFetchingNextPage} onClick={more}>
              Load more
            </Button>
          </div>
        ))}
    </>
  );
}
