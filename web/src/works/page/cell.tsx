import type { FilesCell } from "../../api/gen/model";
import { ManualSearchButton, useStartSearch } from "../../grabs/search";
import { openManualSearch } from "../../manual/sheet-state";
import { Button } from "../../ui/button";
import { Menu } from "../../ui/menu";
import { useShell } from "../../ui/shell-context";
import { StateChip } from "../cell-state";
import { CellDetails } from "./cell-details";
import { CellFacts } from "./cell-facts";
import { type CellView, type MatrixShown, cellScope, fileOf } from "./matrix-model";

/** Search while the cell is wanted, and Manual search; on phone, one menu with both. */
type WorkRef = { readonly id: string; readonly label: string };

function useCellSearch(work: WorkRef, cell: FilesCell) {
  const search = useStartSearch();
  return {
    busy: search.isPending,
    start: () => {
      search.start(work.id, { name: work.label }, { levelIds: [], itemIds: [cell.itemId] });
    },
  };
}

function CellActions({ cell, name, work }: { cell: FilesCell; name: string; work: WorkRef }) {
  const workId = work.id;
  const { busy, start: startSearch } = useCellSearch(work, cell);
  const searchButton = (onClick?: () => void) => (
    <Button
      size="small"
      look="ghost"
      aria-label={`Search for ${name}`}
      busy={busy}
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
        <StateChip state={cell.state} />
        {file !== null && <CellDetails view={view} cell={cell} file={file} shown={shown} />}
      </span>
      <CellFacts cell={cell} grabs={shown.grabs} workId={shown.work.id} />
      <CellActions cell={cell} name={`${view.target.name}, ${row}`} work={shown.work} />
    </div>
  );
}
