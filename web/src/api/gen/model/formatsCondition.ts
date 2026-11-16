import type { FormatsConditionFields } from "./formatsConditionFields.ts";

export interface FormatsCondition {
  /** Each field by name, as Radarr and Sonarr export them, such as value, or min and max; a value is a number in the radarr and sonarr dialects and a name in slipstream's */
  fields: FormatsConditionFields;
  /** Radarr's and Sonarr's implementation name, such as ReleaseTitleSpecification */
  kind: string;
  name: string;
  /** The condition matches when its test fails */
  negate: boolean;
  /** This condition must match for the format to match; without required, any one condition of the same kind is enough */
  required: boolean;
}
