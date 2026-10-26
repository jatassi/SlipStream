import type { FilesCell, GrabsGrab } from "../../api/gen/model";
import { useStartSearch } from "../../grabs/search";
import { Button } from "../../ui/button";
import { stateChips } from "../cell-state";
import { CellDetails } from "./cell-details";
import { CellFacts } from "./cell-facts";
import type { CellView, MatrixShown } from "./matrix-model";

// The latest Grab for the cell: the one its state reads from.
function grabOf(grabs: readonly GrabsGrab[], cell: FilesCell): GrabsGrab | undefined {
  return grabs
    .filter(
      (grab) =>
        grab.itemId === cell.itemId &&
        grab.edition === cell.edition &&
        grab.targetIds.includes(cell.targetId),
    )
    .toSorted((a, b) => a.createdAt.localeCompare(b.createdAt))
    .at(-1);
}

const searchable = new Set<FilesCell["state"]>(["missing", "failed", "upgradable"]);

/** One cell: its state, its file's details, what it adds, and Search while it lacks a file. */
export function CellBlock({
  view,
  row,
  shown,
}: {
  view: CellView;
  row: string;
  shown: MatrixShown;
}) {
  const search = useStartSearch();
  const { cell } = view;
  if (cell === undefined) {
    return null;
  }
  return (
    <div className="flex min-w-0 flex-col items-start gap-1.5">
      <span className="flex items-center gap-1">
        {stateChips[cell.state]}
        {"file" in cell && <CellDetails view={view} cell={cell} shown={shown} />}
      </span>
      <CellFacts cell={cell} grab={grabOf(shown.grabs, cell)} />
      {searchable.has(cell.state) && (
        <Button
          size="small"
          look="ghost"
          aria-label={`Search for ${view.target.name}, ${row}`}
          busy={search.isPending}
          onClick={() => {
            search.mutate({ workId: shown.work.id });
          }}
        >
          Search
        </Button>
      )}
    </div>
  );
}
