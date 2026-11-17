import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { TypeFilter } from "../grabs/jobs";
import { Segmented } from "../ui/segmented";
import { useShell } from "../ui/shell-context";
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

/**
 * The Content Type, the groups and their kinds, the dates and the Work
 * History lists, with History's menu at the end of the line, or of the
 * chips' line on a phone, where the dates take a line of their own.
 */
export function HistoryFilters({ filter, onFilter }: { filter: HistoryFilter; onFilter: Change }) {
  const types = useListContentTypesSuspense().data.contentTypes;
  const phone = useShell() === "phone";
  const chips = <GroupChips filter={filter} onFilter={onFilter} />;
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
      {phone ? (
        <div className="flex w-full items-center gap-2">
          {chips}
          <span className="ml-auto">
            <HistoryMenu />
          </span>
        </div>
      ) : (
        chips
      )}
      <Segmented
        label="When"
        options={whenOptions}
        value={filter.since ?? any}
        onValueChange={(value) => {
          onFilter({ since: value === any ? undefined : value });
        }}
      />
      {filter.work !== undefined && <WorkChip filter={filter} onFilter={onFilter} />}
      {!phone && (
        <span className="ml-auto">
          <HistoryMenu />
        </span>
      )}
    </div>
  );
}
