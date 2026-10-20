import { useState } from "react";

import { Checkbox } from "../ui/checkbox";
import type { Row } from "./model";
import type { Names } from "./names";
import { noPicks, pickedOf, toggle, toggleAll } from "./selection";

export type Picking = {
  readonly has: (id: string) => boolean;
  readonly toggle: (row: Row, range: boolean) => void;
  readonly all: "none" | "some" | "every";
  readonly toggleAll: () => void;
};

export function usePicking(shown: readonly Row[]): {
  picking: Picking;
  picked: Row[];
  clear: () => void;
} {
  const [picks, setPicks] = useState(noPicks);
  const order = shown.map((row) => row.id);
  const picked = pickedOf(picks, shown);
  return {
    picking: {
      has: (id) => picks.ids.has(id),
      toggle: (row, range) => {
        setPicks(toggle(picks, row.id, order, range));
      },
      all: picked.length === 0 ? "none" : picked.length === shown.length ? "every" : "some",
      toggleAll: () => {
        setPicks(toggleAll(picks, order));
      },
    },
    picked,
    clear: () => {
      setPicks(noPicks);
    },
  };
}

export function PickCell({ row, picking, names }: { row: Row; picking: Picking; names: Names }) {
  return (
    <td data-col="pick" className="relative">
      <Checkbox
        label={`Pick ${names.work(row.workId) ?? row.name}`}
        hideLabel
        checked={picking.has(row.id)}
        onCheckedChange={(_, by) => {
          picking.toggle(row, by.shiftKey);
        }}
      />
    </td>
  );
}

export function PickAllCell({ picking }: { picking: Picking }) {
  return (
    <th scope="col" data-col="pick">
      <Checkbox
        label="Pick every shown torrent"
        hideLabel
        checked={picking.all === "every"}
        indeterminate={picking.all === "some"}
        onCheckedChange={picking.toggleAll}
      />
    </th>
  );
}
