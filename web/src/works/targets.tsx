import type {
  FilesCell,
  FilesCellList,
  FilesItemStatusStatus,
  WorksContentType,
  WorksItem,
  WorksTree,
} from "../api/gen/model";
import { useListCutsSuspense } from "../api/gen/query/targets/targets";
import { Section } from "../ui/section";
import { type Column, Table } from "../ui/table";
import { CellState, ItemStatus } from "./cell-state";
import { WantsSheet } from "./wants-sheet";

function editionsOf(cells: readonly FilesCell[]): [string, FilesCell[]][] {
  const editions = new Map<string, FilesCell[]>();
  for (const cell of cells) {
    editions.set(cell.edition, [...(editions.get(cell.edition) ?? []), cell]);
  }
  return [...editions];
}

function columnsOf(
  type: WorksContentType,
  cells: readonly FilesCell[],
  statusOf: (itemId: string) => FilesItemStatusStatus,
): [Column<WorksItem>, ...Column<WorksItem>[]] {
  const targets = [...new Map(cells.map((cell) => [cell.targetId, cell.targetName]))];
  const cellAt = new Map(cells.map((cell) => [`${cell.itemId} ${cell.targetId}`, cell]));
  return [
    { id: "item", header: type.item.one, cell: (item) => item.label },
    { id: "status", header: "Status", cell: (item) => <ItemStatus status={statusOf(item.id)} /> },
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
  cells: { cells, items },
}: {
  type: WorksContentType;
  work: WorksTree;
  cells: FilesCellList;
}) {
  const { cuts } = useListCutsSuspense().data;
  const cutName = (key: string) =>
    key === "any" ? "Any cut" : (cuts.find((cut) => cut.key === key)?.name ?? key);
  const status = new Map(items.map((item) => [item.itemId, item.status]));
  const statusOf = (itemId: string) => status.get(itemId) ?? null;
  return (
    <Section title="Targets">
      <WantsSheet type={type} work={work} />
      {cells.length === 0 ? (
        <p className="text-ink-3">
          Its Profile has no Targets, so this {type.work.one} wants nothing.
        </p>
      ) : (
        <div className="flex flex-col gap-5">
          {editionsOf(cells).map(([key, held]) => (
            <div key={key} className="flex flex-col gap-2">
              <h3 className="text-label text-ink-3">{cutName(key)}</h3>
              <Table
                label={`${cutName(key)} cells`}
                columns={columnsOf(type, held, statusOf)}
                rows={work.items}
                rowKey={(item) => item.id}
                lines={held.some((cell) => "file" in cell) ? 2 : 1}
                stackBelow={560}
              />
            </div>
          ))}
        </div>
      )}
    </Section>
  );
}
