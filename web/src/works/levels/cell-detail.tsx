import type { FilesCell } from "../../api/gen/model";
import { Button } from "../../ui/button";
import { Data } from "../../ui/data";
import { stateChips } from "../cell-state";
import { anyCut } from "../page/matrix-model";
import { CellFacts } from "../page/cell-facts";
import { grabOf } from "../page/cell";
import type { MatrixShown } from "../page/matrix-model";
import { nameOf } from "../paths";
import type { Column, LevelTarget } from "./model";

/** One square's cell, with what names it. */
export type PoppedCell = {
  readonly code: string;
  readonly column: Column;
  readonly target: LevelTarget;
  readonly cell: FilesCell;
};

/** "S01E02 Director's Cut 2160p HDR", or without the cut under any cut. */
export const cellLabel = ({ code, column, target }: Omit<PoppedCell, "cell">) =>
  column.edition === anyCut ? `${code} ${target.name}` : `${code} ${column.name} ${target.name}`;

/**
 * What the cell popover shows of a square: its state, its file or download,
 * or why it failed, and Manual search for the cell.
 */
export function CellDetail({
  popped,
  shown,
  onManualSearch,
}: {
  popped: PoppedCell;
  shown: MatrixShown;
  onManualSearch: () => void;
}) {
  const { cell } = popped;
  return (
    <div className="flex flex-col items-start gap-2">
      <span className="font-semibold text-ink-1">{cellLabel(popped)}</span>
      {stateChips[cell.state]}
      <CellFacts cell={cell} grab={grabOf(shown.grabs, cell)} />
      {"file" in cell && (
        <span className="break-all text-meta text-ink-2">
          <Data>{nameOf(cell.file.path)}</Data>
        </span>
      )}
      <Button size="small" onClick={onManualSearch}>
        Manual search
      </Button>
    </div>
  );
}
