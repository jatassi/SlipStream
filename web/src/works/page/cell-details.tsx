import { InformationCircleIcon } from "@hugeicons/core-free-icons";

import type { FilesCell } from "../../api/gen/model";
import { Data } from "../../ui/data";
import { IconButton } from "../../ui/icon-button";
import { Popover } from "../../ui/popover";
import { folderOf, nameOf } from "../paths";
import type { CellView, MatrixShown } from "./matrix-model";

export function CellDetails({
  view,
  cell,
  shown,
}: {
  view: CellView;
  cell: FilesCell & { file: { id: string; path: string } };
  shown: MatrixShown;
}) {
  const others = (shown.files.find(({ id }) => id === cell.file.id)?.fills ?? []).filter(
    (fill) => fill.targetId !== cell.targetId || fill.edition !== cell.edition,
  );
  return (
    <Popover
      label={`Details of ${view.target.name}`}
      trigger={<IconButton icon={InformationCircleIcon} label={`Details of ${view.target.name}`} />}
    >
      <dl className="flex flex-col gap-2">
        <div>
          <dt className="text-label text-ink-3">File</dt>
          <dd className="break-all text-ink-1">
            <Data>{nameOf(cell.file.path)}</Data>
          </dd>
        </div>
        <div>
          <dt className="text-label text-ink-3">Folder</dt>
          <dd className="break-all text-ink-1">
            <Data>{folderOf(cell.file.path)}</Data>
          </dd>
        </div>
        <div>
          <dt className="text-label text-ink-3">Also fills</dt>
          <dd className="text-ink-1">
            {others.length === 0
              ? "No other cell"
              : others
                  .map(
                    (fill) => `${shown.targetName(fill.targetId)}, ${shown.cutName(fill.edition)}`,
                  )
                  .join("; ")}
          </dd>
        </div>
      </dl>
    </Popover>
  );
}
