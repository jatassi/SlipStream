import type { FilesCell, GrabsGrab } from "../../api/gen/model";
import { ManualSearchButton, useStartSearch } from "../../grabs/search";
import { openManualSearch } from "../../manual/sheet-state";
import { Button } from "../../ui/button";
import { Menu } from "../../ui/menu";
import { useShell } from "../../ui/shell-context";
import { stateChips } from "../cell-state";
import { CellDetails } from "./cell-details";
import { CellFacts } from "./cell-facts";
import { type CellView, type MatrixShown, cellScope } from "./matrix-model";

// The latest Grab for the cell: the one its state reads from.
export function grabOf(grabs: readonly GrabsGrab[], cell: FilesCell): GrabsGrab | undefined {
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

/** Search while the cell lacks a file, and Manual search; on phone, one menu with both. */
function CellActions({ cell, name, workId }: { cell: FilesCell; name: string; workId: string }) {
  const search = useStartSearch();
  const startSearch = () => {
    search.start(workId, { levelIds: [], itemIds: [cell.itemId] });
  };
  const searchButton = (onClick?: () => void) => (
    <Button
      size="small"
      look="ghost"
      aria-label={`Search for ${name}`}
      busy={search.isPending}
      {...(onClick && { onClick })}
    >
      Search
    </Button>
  );
  if (useShell() === "phone") {
    return (
      <Menu
        label={`Actions for ${name}`}
        trigger={searchButton()}
        items={[
          ...(searchable.has(cell.state) ? [{ label: "Search", onSelect: startSearch }] : []),
          {
            label: "Manual search",
            onSelect: () => {
              openManualSearch({ workId, scope: cellScope(cell) });
            },
          },
        ]}
      />
    );
  }
  return (
    <span className="flex flex-wrap items-center gap-1">
      {searchable.has(cell.state) && searchButton(startSearch)}
      <ManualSearchButton
        workId={workId}
        scope={cellScope(cell)}
        label={`Manual search for ${name}`}
        size="small"
        look="ghost"
      />
    </span>
  );
}

/** One cell: its state, its file's details, what it adds, and its actions. */
export function CellBlock({
  view,
  row,
  shown,
}: {
  view: CellView;
  row: string;
  shown: MatrixShown;
}) {
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
      <CellActions cell={cell} name={`${view.target.name}, ${row}`} workId={shown.work.id} />
    </div>
  );
}
