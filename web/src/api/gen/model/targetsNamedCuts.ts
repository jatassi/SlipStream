import type { TargetsNamedCutsKind } from "./targetsNamedCutsKind.ts";

export interface TargetsNamedCuts {
  /**
   * Cut keys in list order
   * @minItems 1
   */
  cuts: string[];
  /** Named cuts: with Standard, each other cut once a release or file of it turns up for an Item; without it, every cut every time */
  kind: TargetsNamedCutsKind;
}
