import { HistoryDays } from "./history-days";
import { HistoryFilters } from "./history-filters";
import { HistorySentence } from "./history-sentence";
import type { HistoryFilter } from "./routed";

const none: Required<{ [K in keyof HistoryFilter]: undefined }> = {
  type: undefined,
  groups: undefined,
  kinds: undefined,
  since: undefined,
  work: undefined,
};

/** What happened to the library: the week in a sentence, the filters, then the entries by day. */
export function HistoryPage({
  filter,
  onFilter,
}: {
  filter: HistoryFilter;
  onFilter: (change: Partial<HistoryFilter>) => void;
}) {
  return (
    <>
      <HistorySentence />
      <HistoryFilters filter={filter} onFilter={onFilter} />
      <HistoryDays
        filter={filter}
        onClear={() => {
          onFilter(none);
        }}
      />
    </>
  );
}
