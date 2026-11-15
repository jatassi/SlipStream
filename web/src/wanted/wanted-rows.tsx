import { useSuspenseInfiniteQuery } from "@tanstack/react-query";
import { useEffect, useEffectEvent, useRef } from "react";

import type { Clock } from "../format";
import { Empty } from "../shell/empty";
import { Button } from "../ui/button";
import { useShell } from "../ui/shell-context";
import { WantedList } from "./wanted-phone";
import { type WantedFilter, wantedQuery } from "./wanted-query";
import { WantedTable } from "./wanted-table";

/** Loads the next page once the end of a phone's list scrolls into view. */
function NextOnScroll({ onNext }: { onNext: () => void }) {
  const end = useRef<HTMLDivElement>(null);
  const next = useEffectEvent(onNext);
  useEffect(() => {
    const watcher = new IntersectionObserver((entries) => {
      if (entries.some((entry) => entry.isIntersecting)) {
        next();
      }
    });
    if (end.current !== null) {
      watcher.observe(end.current);
    }
    return () => {
      watcher.disconnect();
    };
  }, []);
  return <div ref={end} aria-hidden="true" className="h-px" />;
}

/** Every loaded page's rows, a table on desktop ending in Load more, a list on phone that loads as it scrolls. */
export function WantedRows({ filter, clock }: { filter: WantedFilter; clock: Clock }) {
  const phone = useShell() === "phone";
  const { data, hasNextPage, isFetchingNextPage, fetchNextPage } = useSuspenseInfiniteQuery(
    wantedQuery(filter),
  );
  const rows = data.pages.flatMap((page) => page.rows);
  if (rows.length === 0) {
    return <Empty>Nothing is wanted.</Empty>;
  }
  const more = () => {
    if (hasNextPage && !isFetchingNextPage) {
      void fetchNextPage();
    }
  };
  return (
    <>
      {phone ? <WantedList rows={rows} clock={clock} /> : <WantedTable rows={rows} clock={clock} />}
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
