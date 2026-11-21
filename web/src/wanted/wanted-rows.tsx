import { useSuspenseInfiniteQuery } from "@tanstack/react-query";

import type { Clock } from "../format";
import { Button } from "../ui/button";
import { Empty } from "../ui/empty";
import { NextOnScroll } from "../ui/next-on-scroll";
import { useShell } from "../ui/shell-context";
import { WantedList } from "./wanted-phone";
import { type WantedFilter, wantedQuery } from "./wanted-query";
import { WantedTable } from "./wanted-table";
import type { Headers } from "./wanted-words";

/** Every loaded page's rows, a table on desktop ending in Load more, a list on phone that loads as it scrolls. */
export function WantedRows({
  filter,
  headers,
  clock,
}: {
  filter: WantedFilter;
  headers: Headers;
  clock: Clock;
}) {
  const phone = useShell() === "phone";
  const { data, hasNextPage, isFetchingNextPage, fetchNextPage } = useSuspenseInfiniteQuery(
    wantedQuery(filter),
  );
  const rows = data.pages.flatMap((page) => page.rows);
  if (rows.length === 0) {
    return <Empty look="sheet">Nothing to search for.</Empty>;
  }
  const more = () => {
    if (hasNextPage && !isFetchingNextPage) {
      void fetchNextPage();
    }
  };
  return (
    <>
      {phone ? (
        <WantedList rows={rows} clock={clock} />
      ) : (
        <WantedTable rows={rows} headers={headers} clock={clock} />
      )}
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
