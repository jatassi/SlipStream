import type { IndexersDefinitionFieldType } from "./indexersDefinitionFieldType.ts";
import type { IndexersOption } from "./indexersOption.ts";

export interface IndexersDefinitionField {
  /**
   * Null when the field has none
   * @nullable
   */
  default: string | null;
  /**
   * An info field's text, written by the definition's authors
   * @nullable
   */
  help: string | null;
  label: string;
  /**
   * A number's greatest value; null for none
   * @nullable
   */
  max: number | null;
  /**
   * A number's least value; null for none
   * @nullable
   */
  min: number | null;
  /** May hold hyphens or start with a digit */
  name: string;
  /** A select's or a multiselect's choices, by value */
  options: IndexersOption[];
  /** Its saved value is never returned */
  sealed: boolean;
  type: IndexersDefinitionFieldType;
}
