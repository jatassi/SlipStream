import { useSuspenseQuery } from "@tanstack/react-query";
import { useDeferredValue, useState } from "react";

import { getListBlocklistSuspenseQueryOptions } from "../api/gen/query/grabs/grabs";
import { useNow } from "../tasks/use-now";
import { BulkBar } from "./bulk";
import { Nothing } from "./nothing";
import { Pager } from "./pager";
import { usePicks } from "./picks";
import { BlocklistTable } from "./table";
import { Toolbar } from "./toolbar";

export type BlocklistSearch = {
  readonly q?: string | undefined;
  readonly offset?: number | undefined;
};

/**
 * Activity's Blocklist: pages of 50, newest first, filtered by title. The
 * picked entries' Remove keeps the page, and grabs.blocklist-changed refreshes it.
 */
export function BlocklistBrowse({
  search,
  onSearchChange,
}: {
  search: BlocklistSearch;
  onSearchChange: (change: BlocklistSearch) => void;
}) {
  const [draft, setDraft] = useState(search.q ?? "");
  const q = useDeferredValue(draft);
  const offset = useDeferredValue(search.offset ?? 0);
  const { data } = useSuspenseQuery(
    getListBlocklistSuspenseQueryOptions({
      ...(q !== "" && { q }),
      ...(offset > 0 && { offset }),
    }),
  );
  const picks = usePicks(data.entries.map((entry) => entry.infoHash));
  const clock = { now: useNow(60_000), zone: undefined };
  const write = (next: string) => {
    setDraft(next);
    onSearchChange({ q: next === "" ? undefined : next, offset: undefined });
  };
  return (
    <>
      <Toolbar picks={data.entries.length > 0 ? picks : undefined} draft={draft} onDraft={write} />
      {picks.picked.length > 0 && <BulkBar picked={picks.picked} onClear={picks.clear} />}
      {data.total === 0 ? (
        <Nothing
          q={q}
          onClear={() => {
            write("");
          }}
        />
      ) : (
        <BlocklistTable entries={data.entries} picking={picks.picking} clock={clock} />
      )}
      <Pager
        offset={offset}
        shown={data.entries.length}
        total={data.total}
        onOffset={(next) => {
          onSearchChange({ offset: next });
        }}
      />
    </>
  );
}
