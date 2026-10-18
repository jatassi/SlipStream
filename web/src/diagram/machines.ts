import type { Point3, Projection } from "@anatomy/iso-kit";

import type { RoleName } from "../nodes/roles";
import { TOP } from "./layout";
import { board } from "./parts/board";
import { cabinet } from "./parts/cabinet";
import type { Item } from "./parts/part";
import { movedP, placed } from "./parts/placed";
import { press } from "./parts/press";
import { tower } from "./parts/tower";

type Machine = { readonly at: Point3; readonly draw: (P: Projection) => Item[] };

/** The Roles with a machine, and where it stands on the tile's top. */
const MACHINES: Partial<Record<RoleName, Machine>> = {
  downloading: { at: [0, 0, TOP], draw: board },
  storage: { at: [0, 0, TOP], draw: (P) => [cabinet(P)] },
  converting: { at: [-152, -62, TOP], draw: (P) => [press(P)] },
  relaying: { at: [60, -30, TOP], draw: (P) => [tower(P)] },
};

export const hasMachine = (role: RoleName): boolean => MACHINES[role] !== undefined;

/** A Role's machine as the items it sorts by, in tile coordinates; none for a Role without one. */
export function machineOf(P: Projection, role: RoleName): Item[] {
  const machine = MACHINES[role];
  return machine === undefined
    ? []
    : machine.draw(movedP(P, machine.at)).map((part) => placed(part, machine.at));
}
