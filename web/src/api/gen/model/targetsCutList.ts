import type { TargetsCut } from "./targetsCut.ts";

export interface TargetsCutList {
  /** The built-in cuts, Standard first, then each Content Type's by type, then the custom ones by name */
  cuts: TargetsCut[];
}
