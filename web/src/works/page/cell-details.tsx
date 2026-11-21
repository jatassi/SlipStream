import { InformationCircleIcon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import type { FilesCell, FilesLibraryFile } from "../../api/gen/model";
import { openManualSearch } from "../../manual/sheet-state";
import { Button } from "../../ui/button";
import { Data } from "../../ui/data";
import { Fact, Facts } from "../../ui/facts";
import { IconButton } from "../../ui/icon-button";
import { Popover } from "../../ui/popover";
import { folderOf, nameOf } from "../paths";
// oxlint-disable-next-line import/max-dependencies -- a cell's Details name its file and what else it fills, and open Manual search
import { type CellView, type MatrixShown, anyCut, cellScope } from "./matrix-model";

const list = new Intl.ListFormat("en", { type: "conjunction" });

function FileFacts({ path, others }: { path: string; others: readonly string[] }) {
  return (
    <Facts look="inline">
      <Fact label="File">
        <Data>{nameOf(path)}</Data>
      </Fact>
      <Fact label="Folder">
        <Data>{folderOf(path)}</Data>
      </Fact>
      <Fact label="Also used for">
        {others.length === 0 ? "Nothing else" : list.format(others)}
      </Fact>
    </Facts>
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
      <FileFacts path={file.path} others={others} />
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
