import type { FilesCell } from "../../api/gen/model";
import { ManualSearchButton, useStartSearch } from "../../grabs/search";
import { openManualSearch } from "../../manual/sheet-state";
import { Button } from "../../ui/button";
import { Menu } from "../../ui/menu";
import { useShell } from "../../ui/shell-context";
import { stateChips } from "../cell-state";
import { CellDetails } from "./cell-details";
import { CellFacts } from "./cell-facts";
import { type CellView, type MatrixShown, cellScope, fileOf } from "./matrix-model";

/** Search while the cell is wanted, and Manual search; on phone, one menu with both. */
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
          ...(cell.wanted ? [{ label: "Search", onSelect: startSearch }] : []),
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
      {cell.wanted && searchButton(startSearch)}
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
  const file = fileOf(cell);
  return (
    <div className="flex min-w-0 flex-col items-start gap-1.5">
      <span className="flex items-center gap-1">
        {stateChips[cell.state]}
        {file !== null && <CellDetails view={view} cell={cell} file={file} shown={shown} />}
      </span>
      <CellFacts cell={cell} grabs={shown.grabs} />
      <CellActions cell={cell} name={`${view.target.name}, ${row}`} workId={shown.work.id} />
    </div>
  );
}
