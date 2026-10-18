import type { TargetsNamedCutsKind } from "./targetsNamedCutsKind.ts";

export interface TargetsNamedCuts {
  /**
   * Cut keys in list order
   * @minItems 1
   */
  cuts: string[];
  /** Named cuts: with Standard, each other cut once seen for an Item; without it, every cut outright */
  kind: TargetsNamedCutsKind;
}
