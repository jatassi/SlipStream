import type { GrabsChoice } from "./grabsChoice.ts";
import type { GrabsPlanPick } from "./grabsPlanPick.ts";

export interface GrabsPlanRow {
  /** For a cell nothing fills, its plain cause, such as No Indexer listed a 720p Release. */
  cause?: string;
  /** Every other Release that could fill its first cell, ranked */
  choices: GrabsChoice[];
  /** For a failed cell, its last Grab's failure */
  failure?: string;
  /** The row's first cell, stable while that cell leads it. A pin chosen from its choices replaces the pin with this key. */
  key: string;
  /** For a cell nothing fills, such as Missing: 720p for E05. */
  missing?: string;
  /** What a Grab of it gets, such as It gets you 1080p and 720p for E05. */
  outcome?: string;
  /** Null for a cell nothing fills */
  pick: GrabsPlanPick | null;
  /** Its Release was swapped in by a pin */
  pinned: boolean;
}
