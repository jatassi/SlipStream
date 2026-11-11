import type { GrabsHeldCell } from "./grabsHeldCell.ts";
import type { GrabsPlanRow } from "./grabsPlanRow.ts";

export interface GrabsPlan {
  headline: string;
  /** The cells a Grab downloads, which the plan leaves out */
  held: GrabsHeldCell[];
  /** Each Grab and each cell nothing fills, in Want order */
  rows: GrabsPlanRow[];
}
