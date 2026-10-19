import type { Point3, Projection } from "@anatomy/iso-kit";

import type { RoleName } from "../nodes/roles";
import type { NodeRow } from "../routes/_app/nodes/-landing/view";
import { TOP } from "./layout";
import type { Item } from "./parts/part";
import { type Lead, lead } from "./parts/lead";
import { movedP, placed } from "./parts/placed";
import { SMALL } from "./parts/small";
import { TALL } from "./parts/tall";

type Machine = {
  /** Where it stands on the tile's top, the same on every tile. */
  readonly at: Point3;
  /** It moves with the Role's real work, rather than standing at rest. */
  readonly live: boolean;
  /**
   * Its cable into the tile, in its own coordinates; absent when the machine
   * draws its cable among its own parts.
   */
  readonly lead?: Lead;
};

const DRAW: Readonly<Record<RoleName, (P: Projection, node: NodeRow) => Item[]>> = {
  ...TALL,
  ...SMALL,
};

// The tall machines stand along the back; the small ones in two rows in
// front of them, clear of the chips.
const MACHINES: Readonly<Record<RoleName, Machine>> = {
  searching: {
    at: [18, -26, TOP],
    live: true,
    lead: { from: [3, 9, 5], toward: [0, 1], reach: 10 },
  },
  downloading: {
    at: [0, 0, TOP],
    live: true,
    lead: { from: [-55, -86, 4.2], toward: [0, 1], reach: 9 },
  },
  storage: { at: [-34, 0, TOP], live: true },
  converting: {
    at: [-152, -62, TOP],
    live: false,
    lead: { from: [102, 94, 6.5], toward: [0, 1], reach: 9 },
  },
  subtitles: {
    at: [66, -26, TOP],
    live: false,
    lead: { from: [-18, 9, 5], toward: [-1, 0], reach: 9 },
  },
  metadata: {
    at: [32, 44, TOP],
    live: true,
    lead: { from: [-17, 8, 6], toward: [-1, 0], reach: 9 },
  },
  "media-server-link": {
    at: [130, -24, TOP],
    live: false,
    lead: { from: [-14, 5, 16], toward: [-1, 0], reach: 10 },
  },
  notifying: {
    at: [78, 46, TOP],
    live: false,
    lead: { from: [-5.4, 4, 32], toward: [-1, 0], reach: 7 },
  },
  relaying: {
    at: [100, -110, TOP],
    live: false,
    lead: { from: [52, 44, 9], toward: [0, 1], reach: 8 },
  },
  serving: {
    at: [124, 40, TOP],
    live: true,
    lead: { from: [-14.5, 6, 6.5], toward: [-1, 0], reach: 10 },
  },
};

/** Whether a Role's machine moves with its real work; the rest stand at rest. */
export const movesWithWork = (role: RoleName): boolean => MACHINES[role].live;

export const machineP = (P: Projection, role: RoleName): Projection => movedP(P, MACHINES[role].at);

/** A Role's machine on a Node's tile and its cable into the tile, as the items it sorts by, in tile coordinates. */
export function machineOf(P: Projection, node: NodeRow, role: RoleName): Item[] {
  const machine = MACHINES[role];
  const local = machineP(P, role);
  const parts = DRAW[role](local, node);
  const drawn =
    machine.lead === undefined ? parts : [...parts, lead(local, `${role}.lead`, machine.lead)];
  return drawn.map((part) => placed(part, machine.at));
}
