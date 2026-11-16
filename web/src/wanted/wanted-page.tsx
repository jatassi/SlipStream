import { useSuspenseInfiniteQuery } from "@tanstack/react-query";

import type { WorksContentType } from "../api/gen/model";
import type { Clock } from "../format";
import { Page } from "../shell/page";
import { useNow } from "../tasks/use-now";
import { StatusSentence } from "../ui/status-sentence";
import { CancelQueued, SearchAll } from "./wanted-actions";
import { type WantedFilter, wantedQuery } from "./wanted-query";
import { WantedRows } from "./wanted-rows";
import { ReasonChips, SortMenu, TypeLinks } from "./wanted-toolbar";
// oxlint-disable-next-line import/max-dependencies -- the page lays out the status, the filters, its actions and the rows
import { headersOf, statusParts } from "./wanted-words";

const minuteMs = 60_000;

/**
 * What SlipStream is searching for: the queue's status, then one row per
 * Work that wants a Release, filtered by reason and sorted, 100 at a time.
 */
export function WantedPage({
  types,
  filter,
  onFilter,
}: {
  types: readonly WorksContentType[];
  filter: WantedFilter;
  onFilter: (next: Partial<WantedFilter>) => void;
}) {
  const clock: Clock = { now: useNow(minuteMs), zone: undefined };
  const [first] = useSuspenseInfiniteQuery(wantedQuery(filter)).data.pages;
  if (first === undefined) {
    throw new Error("Wanted loaded no page");
  }
  const type = types.find(({ id }) => id === filter.contentType);
  return (
    <Page
      beside={types.length < 2 ? undefined : <TypeLinks types={types} />}
      action={<SearchAll filter={filter} type={type} total={first.total} />}
    >
      <div className="mb-4">
        <StatusSentence parts={statusParts(first.status, clock)} />
      </div>
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <ReasonChips
          chosen={filter.reasons}
          onChange={(reasons) => {
            onFilter({ reasons });
          }}
        />
        <SortMenu
          sort={filter.sort}
          onSort={(sort) => {
            onFilter({ sort });
          }}
        />
        {first.status.queued > 0 && <CancelQueued />}
      </div>
      <WantedRows
        filter={filter}
        headers={headersOf(type === undefined ? types : [type])}
        clock={clock}
      />
    </Page>
  );
}
