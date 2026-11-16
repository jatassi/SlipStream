import type { GrabsHeldCell } from "./grabsHeldCell.ts";
import type { GrabsPlanRow } from "./grabsPlanRow.ts";

export interface GrabsPlan {
  headline: string;
  /** The Items and Targets a Grab is already downloading, which the plan leaves out */
  held: GrabsHeldCell[];
  /** Each Grab and each Item and Target no release covers, in the order the plan walks them */
  rows: GrabsPlanRow[];
}
