import type { Point3, Projection } from "@anatomy/iso-kit";

import type { RoleName } from "../nodes/roles";
import type { NodeRow } from "../routes/_app/nodes/-landing/view";
import { TOP } from "./layout";
import type { Item } from "./parts/part";
import { movedP, placed } from "./parts/placed";
import { SMALL } from "./parts/small";
import { TALL } from "./parts/tall";

type Machine = {
  /** Where it stands on the tile's top, the same on every tile. */
  readonly at: Point3;
  /** It moves with the Role's real work, rather than standing at rest. */
  readonly live: boolean;
};

const DRAW: Readonly<Record<RoleName, (P: Projection, node: NodeRow) => Item[]>> = {
  ...TALL,
  ...SMALL,
};

// The tall machines stand along the back; the small ones in two rows in
// front of them, clear of the chips.
const MACHINES: Readonly<Record<RoleName, Machine>> = {
  searching: { at: [18, -26, TOP], live: true },
  downloading: { at: [0, 0, TOP], live: true },
  storage: { at: [-34, 0, TOP], live: false },
  converting: { at: [-152, -62, TOP], live: false },
  subtitles: { at: [66, -26, TOP], live: false },
  metadata: { at: [32, 44, TOP], live: true },
  "media-server-link": { at: [130, -24, TOP], live: false },
  notifying: { at: [78, 46, TOP], live: false },
  relaying: { at: [100, -110, TOP], live: false },
  serving: { at: [124, 40, TOP], live: true },
};

/** Whether a Role's machine moves with its real work; the rest stand at rest. */
export const movesWithWork = (role: RoleName): boolean => MACHINES[role].live;

export const machineP = (P: Projection, role: RoleName): Projection => movedP(P, MACHINES[role].at);

/** A Role's machine on a Node's tile, as the items it sorts by, in tile coordinates. */
export function machineOf(P: Projection, node: NodeRow, role: RoleName): Item[] {
  const { at } = MACHINES[role];
  return DRAW[role](machineP(P, role), node).map((part) => placed(part, at));
}
