import type { GrabsCellKey } from "./grabsCellKey.ts";

export interface GrabsWaiting {
  /**
   * The Releases it holds
   * @minimum 1
   */
  candidates: number;
  cell: GrabsCellKey;
  /** When the wait ends and SlipStream grabs the best Release it holds */
  until: string;
}
