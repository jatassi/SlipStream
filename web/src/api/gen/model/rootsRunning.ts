import type { RootsRunningPhase } from "./rootsRunningPhase.ts";

export interface RootsRunning {
  /**
   * The files listed, compared or probed so far
   * @minimum 0
   */
  done: number;
  /** listing: reading the folder, total null; comparing: matching the listed files with the records; probing: reading media info */
  phase: RootsRunningPhase;
  /** The run Cancel library scan cancels */
  runId: number;
  /**
   * The files to compare or probe; null while listing
   * @minimum 0
   * @nullable
   */
  total: number | null;
}
