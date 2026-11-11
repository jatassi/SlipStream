import { InformationCircleIcon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import type { FilesCell } from "../../api/gen/model";
import { openManualSearch } from "../../manual/sheet-state";
import { Button } from "../../ui/button";
import { Data } from "../../ui/data";
import { IconButton } from "../../ui/icon-button";
import { Popover } from "../../ui/popover";
import { folderOf, nameOf } from "../paths";
import { type CellView, type MatrixShown, cellScope } from "./matrix-model";

function Facts({ path, others }: { path: string; others: readonly string[] }) {
  return (
    <dl className="flex flex-col gap-2">
      <div>
        <dt className="text-label text-ink-3">File</dt>
        <dd className="break-all text-ink-1">
          <Data>{nameOf(path)}</Data>
        </dd>
      </div>
      <div>
        <dt className="text-label text-ink-3">Folder</dt>
        <dd className="break-all text-ink-1">
          <Data>{folderOf(path)}</Data>
        </dd>
      </div>
      <div>
        <dt className="text-label text-ink-3">Also fills</dt>
        <dd className="text-ink-1">{others.length === 0 ? "No other cell" : others.join("; ")}</dd>
      </div>
    </dl>
  );
}

export function CellDetails({
  view,
  cell,
  shown,
}: {
  view: CellView;
  cell: FilesCell & { file: { id: string; path: string } };
  shown: MatrixShown;
}) {
  const [open, setOpen] = useState(false);
  const others = (shown.files.find(({ id }) => id === cell.file.id)?.fills ?? [])
    .filter((fill) => fill.targetId !== cell.targetId || fill.edition !== cell.edition)
    .map((fill) => `${shown.targetName(fill.targetId)}, ${shown.cutName(fill.edition)}`);
  return (
    <Popover
      label={`Details of ${view.target.name}`}
      open={open}
      onOpenChange={setOpen}
      trigger={<IconButton icon={InformationCircleIcon} label={`Details of ${view.target.name}`} />}
    >
      <Facts path={cell.file.path} others={others} />
      <Button
        size="small"
        className="mt-3"
        onClick={() => {
          setOpen(false);
          openManualSearch({ workId: shown.work.id, scope: cellScope(cell) });
        }}
      >
        Manual search
      </Button>
    </Popover>
  );
}
