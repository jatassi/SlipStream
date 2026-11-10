import type { GrabsManualSearch, GrabsPinned } from "../api/gen/model";
import { useStartManualSearch } from "../api/gen/query/grabs/grabs";
import { sentenceOf } from "../api/errors";
import { AlertLine } from "../ui/alert-line";
import { Plan } from "./plan";
import { ReadingProvider } from "./reading";
import { QueryField, SearchingLine } from "./searching-line";
import type { ManualScope } from "./sheet-state";

/**
 * One Manual search's sheet, top to bottom: the query for a scope of one
 * Item, the Searching line, the headline, then the plan. Searching again
 * starts a new Manual search for the same scope, and its event refreshes
 * the sheet.
 */
export function ManualBody({
  workId,
  scope,
  search,
  onPins,
  onReopen,
  onClose,
}: {
  workId: string;
  scope: ManualScope;
  search: GrabsManualSearch;
  onPins: (pins: readonly GrabsPinned[]) => void;
  onReopen: () => void;
  onClose: () => void;
}) {
  const restart = useStartManualSearch();
  const onRestart = (query: string | null) => {
    restart.mutate({ workId, data: { ...scope, ...(query !== null && { query }), again: true } });
  };
  return (
    <ReadingProvider workId={workId} title={search.title}>
      <div className="flex flex-col gap-4">
        {search.queryable && (
          <QueryField query={search.query} busy={restart.isPending} onRestart={onRestart} />
        )}
        <SearchingLine
          search={search}
          busy={restart.isPending}
          onRestart={onRestart}
          onLeave={onClose}
        />
        {restart.error !== null && <AlertLine text={sentenceOf(restart.error)} />}
        <p className="text-sentence text-ink-1 in-data-[density=touch]:text-sentence-touch">
          {search.plan.headline}
        </p>
        <Plan workId={workId} search={search} onPins={onPins} onReopen={onReopen} />
      </div>
    </ReadingProvider>
  );
}
