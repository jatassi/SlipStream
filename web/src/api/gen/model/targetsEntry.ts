import type { TargetsEntrySourcing } from "./targetsEntrySourcing.ts";

export interface TargetsEntry {
  /** For derive and download_then_derive: the Target above it whose file it is made from */
  fromTargetId?: string;
  /**
   * For download_then_derive: how long SlipStream searches for a download before it makes the file
   * @minimum 3600
   * @maximum 7776000
   */
  graceSeconds?: number;
  /** How the Target gets its file: download, made from another Target's file (derive), or downloaded, or made from another Target's file after a wait (download_then_derive) */
  sourcing: TargetsEntrySourcing;
  targetId: string;
}
