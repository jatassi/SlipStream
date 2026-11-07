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
  /** May hold hyphens or start with a digit */
  name: string;
  /** A select's choices, by value */
  options: IndexersOption[];
  /** Its saved value is never returned */
  sealed: boolean;
  type: IndexersDefinitionFieldType;
}
