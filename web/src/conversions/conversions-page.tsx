import { useSuspenseInfiniteQuery } from "@tanstack/react-query";
import { useDeferredValue, useState } from "react";

import { useClock } from "../ui/clock";
import { StatusSentence } from "../ui/status-sentence";
import { Bands } from "./bands";
import { type Filter, conversionPages, paramsOf } from "./load";
import { Toolbar } from "./toolbar";
import { sentenceParts } from "./words";

/**
 * Activity's Conversions: the sentence, then the failed ones, the running ones
 * and Up next by lane. `search` is Activity's shared filter plus the band.
 */
export function ConversionsPage({
  search,
  onSearchChange,
}: {
  search: Filter;
  onSearchChange: (change: Partial<Filter>) => void;
}) {
  const clock = useClock();
  const [draft, setDraft] = useState(search.q ?? "");
  const q = useDeferredValue(draft);
  const { data, hasNextPage, isFetchingNextPage, fetchNextPage } = useSuspenseInfiniteQuery(
    conversionPages(paramsOf({ ...search, q })),
  );
  const [first] = data.pages;
  if (first === undefined) {
    throw new Error("The Conversions list has no first page");
  }
  return (
    <>
      <StatusSentence parts={sentenceParts(first.counts, clock)} />
      <Toolbar
        counts={first.counts}
        search={search}
        draft={draft}
        onDraft={(next) => {
          setDraft(next);
          onSearchChange({ q: next === "" ? undefined : next });
        }}
        onChange={onSearchChange}
      />
      <Bands
        first={first}
        pages={data.pages}
        more={{
          has: hasNextPage,
          busy: isFetchingNextPage,
          next: () => {
            void fetchNextPage();
          },
        }}
      />
    </>
  );
}
