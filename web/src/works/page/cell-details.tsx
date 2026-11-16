import { InformationCircleIcon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import type { FilesCell, FilesLibraryFile } from "../../api/gen/model";
import { openManualSearch } from "../../manual/sheet-state";
import { Button } from "../../ui/button";
import { Data } from "../../ui/data";
import { IconButton } from "../../ui/icon-button";
import { Popover } from "../../ui/popover";
import { folderOf, nameOf } from "../paths";
import { type CellView, type MatrixShown, anyCut, cellScope } from "./matrix-model";

const list = new Intl.ListFormat("en", { type: "conjunction" });

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
        <dt className="text-label text-ink-3">Also used for</dt>
        <dd className="text-ink-1">{others.length === 0 ? "Nothing else" : list.format(others)}</dd>
      </div>
    </dl>
  );
}

export function CellDetails({
  view,
  cell,
  file,
  shown,
}: {
  view: CellView;
  cell: FilesCell;
  file: FilesLibraryFile;
  shown: MatrixShown;
}) {
  const [open, setOpen] = useState(false);
  const others = (shown.files.find(({ id }) => id === file.id)?.fills ?? [])
    .filter((fill) => fill.targetId !== cell.targetId || fill.edition !== cell.edition)
    .map((fill) =>
      fill.edition === anyCut
        ? shown.targetName(fill.targetId)
        : `${shown.cutName(fill.edition)} ${shown.targetName(fill.targetId)}`,
    );
  return (
    <Popover
      label={`Details of ${view.target.name}`}
      open={open}
      onOpenChange={setOpen}
      trigger={<IconButton icon={InformationCircleIcon} label={`Details of ${view.target.name}`} />}
    >
      <Facts path={file.path} others={others} />
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
