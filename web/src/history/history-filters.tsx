import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { TypeFilter } from "../grabs/jobs";
import { Segmented } from "../ui/segmented";
import type { SinceChip } from "./day-heading";
import { GroupChips, WorkChip } from "./filter-chips";
import { HistoryMenu } from "./history-menu";
import type { HistoryFilter } from "./routed";

const any = "any";

const whenOptions: readonly { value: SinceChip | typeof any; label: string }[] = [
  { value: any, label: "Any time" },
  { value: "today", label: "Today" },
  { value: "7d", label: "Last 7 days" },
  { value: "30d", label: "Last 30 days" },
];

type Change = (change: Partial<HistoryFilter>) => void;

/** The Content Type, the groups and their kinds, the dates and the Work History lists, then History's menu. */
export function HistoryFilters({ filter, onFilter }: { filter: HistoryFilter; onFilter: Change }) {
  const types = useListContentTypesSuspense().data.contentTypes;
  return (
    <div className="mb-4 flex flex-wrap items-center gap-2">
      {(types.length > 1 || filter.type !== undefined) && (
        <TypeFilter
          types={types}
          type={filter.type}
          onTypeChange={(type) => {
            onFilter({ type });
          }}
        />
      )}
      <GroupChips filter={filter} onFilter={onFilter} />
      <Segmented
        label="When"
        options={whenOptions}
        value={filter.since ?? any}
        onValueChange={(value) => {
          onFilter({ since: value === any ? undefined : value });
        }}
      />
      {filter.work !== undefined && <WorkChip filter={filter} onFilter={onFilter} />}
      <span className="ml-auto">
        <HistoryMenu />
      </span>
    </div>
  );
}
