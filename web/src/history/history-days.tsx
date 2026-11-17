import { useSuspenseInfiniteQuery } from "@tanstack/react-query";

import type { HistoryListEntry } from "../api/gen/model";
import { Empty } from "../shell/empty";
import { useNow } from "../tasks/use-now";
import { Button } from "../ui/button";
import { List } from "../ui/list";
import { NextOnScroll } from "../ui/next-on-scroll";
import { useShell } from "../ui/shell-context";
import { dayHeading, localDayKey } from "./day-heading";
import { historyPages } from "./history-query";
// oxlint-disable-next-line import/max-dependencies -- the list lays out its days, its empty states and its paging
import { HistoryRow } from "./history-row";
import { type HistoryFilter, filtered, paramsOf } from "./routed";

const minuteMs = 60_000;

type Day = { readonly key: string; readonly heading: string; readonly entries: HistoryListEntry[] };

/** entries, newest first, in runs of one browser day each. */
function daysOf(entries: readonly HistoryListEntry[], now: Date): Day[] {
  const days: Day[] = [];
  for (const entry of entries) {
    const key = localDayKey(entry.at);
    const last = days.at(-1);
    if (last?.key === key) {
      last.entries.push(entry);
    } else {
      days.push({ key, heading: dayHeading(entry.at, now), entries: [entry] });
    }
  }
  return days;
}

/** Every loaded page's entries under their day's heading, ending in Load more on desktop and loading as a phone scrolls. */
export function HistoryDays({ filter, onClear }: { filter: HistoryFilter; onClear: () => void }) {
  const now = useNow(minuteMs);
  const phone = useShell() === "phone";
  const { data, hasNextPage, isFetchingNextPage, fetchNextPage } = useSuspenseInfiniteQuery(
    historyPages(paramsOf(filter, now)),
  );
  const entries = data.pages.flatMap((page) => page.entries);
  if (entries.length === 0) {
    return filtered(filter) ? (
      <Empty action={<Button onClick={onClear}>Clear filters</Button>}>
        No entries match these filters.
      </Empty>
    ) : (
      <Empty>
        History is empty. Grabs, Imports and Works added or deleted show here as they happen.
      </Empty>
    );
  }
  const more = () => {
    if (hasNextPage && !isFetchingNextPage) {
      void fetchNextPage();
    }
  };
  return (
    <>
      {daysOf(entries, now).map(({ key, heading, entries: onDay }) => (
        <section key={key}>
          <h3 className="mt-5 mb-2 text-meta font-semibold text-ink-2">{heading}</h3>
          <List label={heading}>
            {onDay.map((entry) => (
              <HistoryRow key={entry.id} entry={entry} />
            ))}
          </List>
        </section>
      ))}
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
