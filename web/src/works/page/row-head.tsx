import { MoreHorizontalIcon } from "@hugeicons/core-free-icons";

import type { TargetsEditions } from "../../api/gen/model";
import { IconButton } from "../../ui/icon-button";
import { Menu, type MenuEntry } from "../../ui/menu";
import { editionsChange, editionsSource } from "./editions";
import type { MatrixRow, MatrixShown } from "./matrix-model";

// The Work's own list after the Admin stops wanting one cut or moves it.
function editionEntries(row: MatrixRow, shown: MatrixShown): MenuEntry[] {
  const { editions } = shown.wants;
  if (editions.kind === "any") {
    return [];
  }
  const { cuts } = editions;
  const at = cuts.indexOf(row.edition);
  const saving = (next: string[]) => () => {
    const list: TargetsEditions = { kind: "cuts", cuts: next };
    void shown.save(editionsChange({ work: shown.work, editions: list }));
  };
  const moved = (to: number) => {
    const next = cuts.filter((key) => key !== row.edition);
    next.splice(to, 0, row.edition);
    return next;
  };
  return [
    ...(at > 0 ? [{ label: "Move up", onSelect: saving(moved(at - 1)) }] : []),
    ...(at < cuts.length - 1 ? [{ label: "Move down", onSelect: saving(moved(at + 1)) }] : []),
    ...(cuts.length > 1
      ? [
          {
            label: `Stop wanting ${row.name} on this Work`,
            tone: "danger" as const,
            onSelect: saving(cuts.filter((key) => key !== row.edition)),
          },
        ]
      : []),
  ];
}

/** The Edition's name, where its list comes from, and moving or dropping it in the Work's own list. */
export function RowHead({ row, shown }: { row: MatrixRow; shown: MatrixShown }) {
  const entries = editionEntries(row, shown);
  return (
    <div className="flex items-start justify-between gap-2">
      <div className="flex min-w-0 flex-col">
        <span className="font-semibold text-ink-1">{row.name}</span>
        <span className="text-meta text-ink-3">{editionsSource(shown.wants)}</span>
      </div>
      {entries.length > 0 && (
        <Menu
          label={`Actions for ${row.name}`}
          trigger={<IconButton icon={MoreHorizontalIcon} label={`Actions for ${row.name}`} />}
          items={entries}
        />
      )}
    </div>
  );
}
