import type { GrabsChoice } from "./grabsChoice.ts";
import type { GrabsPlanPick } from "./grabsPlanPick.ts";

export interface GrabsPlanRow {
  /** For an Item and Target no release covers, its plain cause, such as No Indexer listed a 720p release. */
  cause?: string;
  /** Every other release that could download its first Item and Target, ranked */
  choices: GrabsChoice[];
  /** For a failed Item and Target, its last Grab's failure */
  failure?: string;
  /** The row's first Item and Target, stable while it leads the row. A pin chosen from its choices replaces the pin with this key. */
  key: string;
  /** For an Item and Target no release covers, such as Missing: 720p for E05. */
  missing?: string;
  /** What a Grab of it gets, such as It gets you 1080p and 720p for E05. */
  outcome?: string;
  /** Null for an Item and Target no release covers */
  pick: GrabsPlanPick | null;
  /** Its Release was swapped in by a pin */
  pinned: boolean;
}
