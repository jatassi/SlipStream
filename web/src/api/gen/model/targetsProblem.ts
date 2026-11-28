import type { TargetsProblemCode } from "./targetsProblemCode.ts";
import type { TargetsProblemFacts } from "./targetsProblemFacts.ts";

export interface TargetsProblem {
  code: TargetsProblemCode;
  /** What its words name, by key */
  facts: TargetsProblemFacts;
  /** The Conversion setting it is about, such as codec */
  field: string;
}
