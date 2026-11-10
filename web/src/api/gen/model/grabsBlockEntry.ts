import type { GrabsBlockEntryCause } from "./grabsBlockEntryCause.ts";
import type { GrabsWorkRef } from "./grabsWorkRef.ts";

export interface GrabsBlockEntry {
  addedAt: string;
  /** The cause in a sentence, such as TorrentLeech reported it unregistered. */
  because: string;
  cause: GrabsBlockEntryCause;
  /**
   * The name of the Indexer that listed it; null when none is known
   * @nullable
   */
  indexer: string | null;
  /** @pattern ^[0-9a-f]{40}$ */
  infoHash: string;
  title: string;
  /** What SlipStream reads from the title, in words */
  words: string;
  /** The Work it was grabbed for; null once the Work is deleted */
  work: GrabsWorkRef | null;
}
