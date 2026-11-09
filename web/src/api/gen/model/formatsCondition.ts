import type { FormatsConditionFields } from "./formatsConditionFields.ts";

export interface FormatsCondition {
  /** Each field by name, as Radarr and Sonarr export them, such as value, or min and max; a value is a number in the radarr and sonarr dialects and a name in slipstream's */
  fields: FormatsConditionFields;
  /** Radarr's and Sonarr's implementation name, such as ReleaseTitleSpecification */
  kind: string;
  name: string;
  /** The condition matches when its test fails */
  negate: boolean;
  /** The format matches only when this condition does, beside the others of its kind */
  required: boolean;
}
