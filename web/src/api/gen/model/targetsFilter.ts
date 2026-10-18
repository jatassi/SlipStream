import type { TargetsFilterAttribute } from "./targetsFilterAttribute.ts";
import type { TargetsFilterMode } from "./targetsFilterMode.ts";

export interface TargetsFilter {
  attribute: TargetsFilterAttribute;
  /** Only passes a fact holding one of the values, not one holding none; an unknown fact passes */
  mode: TargetsFilterMode;
  /**
   * The attribute's values by name; a language is an ISO 639 code or original
   * @minItems 1
   */
  values: string[];
}
