import type { TargetsMadeRow } from "./targetsMadeRow.ts";

export interface TargetsRemakeImpact {
  /**
   * Made files the Target's older settings made
   * @minimum 0
   */
  older: number;
  rows: TargetsMadeRow[];
}
