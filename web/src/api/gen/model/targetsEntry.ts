import type { TargetsEntrySourcing } from "./targetsEntrySourcing.ts";

export interface TargetsEntry {
  /** How the Target gets its file */
  sourcing: TargetsEntrySourcing;
  targetId: string;
}
