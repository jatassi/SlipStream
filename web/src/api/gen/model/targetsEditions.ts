import type { TargetsEditionsKind } from "./targetsEditionsKind.ts";

export interface TargetsEditions {
  /** Any cut: one file per Target, of whatever cut */
  kind: TargetsEditionsKind;
}
