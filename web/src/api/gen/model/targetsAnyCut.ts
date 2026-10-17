import type { TargetsAnyCutKind } from "./targetsAnyCutKind.ts";

export interface TargetsAnyCut {
  /** Any cut: one file per Target, of whatever cut */
  kind: TargetsAnyCutKind;
}
