import type { TargetsFilterAttribute } from "./targetsFilterAttribute.ts";
import type { TargetsFilterMode } from "./targetsFilterMode.ts";

export interface TargetsFilter {
  attribute: TargetsFilterAttribute;
  /** only passes a fact that has one of the values, and not passes a fact that has none of them; an unknown fact passes either mode */
  mode: TargetsFilterMode;
  /**
   * The attribute's values by name; a language is an ISO 639 code or original
   * @minItems 1
   */
  values: string[];
}
