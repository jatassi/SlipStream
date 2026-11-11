import type { GrabsEditionOverride } from "./grabsEditionOverride.ts";
import type { GrabsHeldCell } from "./grabsHeldCell.ts";
import type { GrabsWordedReason } from "./grabsWordedReason.ts";

export interface GrabsGrabOffer {
  /** The Grabs downloading a cell it would also fill */
  cancels: GrabsHeldCell[];
  /** The cells a Grab of it alone fills, any a Grab downloads among them. Send them back as the pick's cells. */
  cells: string[];
  /** The Edition it is grabbed as; null unless that is another Edition than its cut's */
  edition: GrabsEditionOverride | null;
  /** It fills some cell and its Indexer would take it now. One on the Blocklist is grabbed only with unblock. */
  grabbable: boolean;
  lifts: GrabsWordedReason[];
  /** What a Grab of it alone gets */
  outcome?: string;
  /** Why it can't be grabbed */
  refusal?: string;
  /** Why its Indexer won't take it now, such as TorrentLeech is at its grab limit until 14:05. */
  waits?: string;
}
