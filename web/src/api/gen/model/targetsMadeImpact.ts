import type { TargetsMadeRow } from "./targetsMadeRow.ts";

export interface TargetsMadeImpact {
  /**
   * About how much they take on disk
   * @minimum 0
   */
  bytes: number;
  /**
   * Files the save has SlipStream make
   * @minimum 0
   */
  files: number;
  /**
   * The Works the backfill's rest moves to a Profile that matches without the made-from Target
   * @minimum 0
   */
  matching: number;
  /** The Nodes forecast to make them */
  nodes: string[];
  rows: TargetsMadeRow[];
  /**
   * About how long they take to make
   * @minimum 0
   */
  seconds: number;
  /** The Works no Node can convert today */
  unconvertible: string[];
  /** @minimum 0 */
  works: number;
}
