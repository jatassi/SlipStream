import type { FormatsCondition } from "./formatsCondition.ts";

export interface FormatsNewFormat {
  /** Conditions of a kind match when one of them does and every required one does; the format matches when every kind's do */
  conditions: FormatsCondition[];
  /** The Custom Formats naming token lists it */
  includeWhenRenaming?: boolean;
  /**
   * Unique among Custom Formats, ignoring case
   * @minLength 1
   * @maxLength 100
   */
  name: string;
}
