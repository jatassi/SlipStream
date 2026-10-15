import type { FilesCell, WorksContentType, WorksItem, WorksTree } from "../api/gen/model";
import { Chip } from "../ui/chip";
import { Data } from "../ui/data";
import { LiveChip } from "../ui/live-chip";
import { MetPill } from "../ui/met-pill";
import { Section } from "../ui/section";
import { SignalChip } from "../ui/signal-chip";
import { type Column, Table } from "../ui/table";
import { nameOf } from "./paths";

function CellState({ cell }: { cell: FilesCell | undefined }) {
  if (cell === undefined) {
    return null;
  }
  if (cell.state === "met") {
    return (
      <span className="flex min-w-0 flex-col items-start gap-1">
        <MetPill>Met</MetPill>
        <span className="max-w-full truncate text-meta text-ink-3">
          <Data>{nameOf(cell.file.path)}</Data>
        </span>
      </span>
    );
  }
  if (cell.state === "downloading") {
    return <LiveChip>Downloading</LiveChip>;
  }
  if (cell.state === "failed") {
    return <SignalChip kind="fail">Failed</SignalChip>;
  }
  return <Chip>Missing</Chip>;
}

function columnsOf(
  type: WorksContentType,
  cells: readonly FilesCell[],
): [Column<WorksItem>, ...Column<WorksItem>[]] {
  const targets = [...new Map(cells.map((cell) => [cell.targetId, cell.targetName]))];
  const cellAt = new Map(cells.map((cell) => [`${cell.itemId} ${cell.targetId}`, cell]));
  return [
    { id: "item", header: type.item.one, cell: (item) => item.label },
    ...targets.map(([id, name]): Column<WorksItem> => ({
      id,
      header: name,
      cell: (item) => <CellState cell={cellAt.get(`${item.id} ${id}`)} />,
    })),
  ];
}

export function Targets({
  type,
  work,
  cells,
}: {
  type: WorksContentType;
  work: WorksTree;
  cells: readonly FilesCell[];
}) {
  return (
    <Section title="Targets">
      <Table
        label="Targets"
        columns={columnsOf(type, cells)}
        rows={work.items}
        rowKey={(item) => item.id}
        lines={cells.some((cell) => cell.state === "met") ? 2 : 1}
      />
    </Section>
  );
}
