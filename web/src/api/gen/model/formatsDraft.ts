import type { FormatsCondition } from "./formatsCondition.ts";

export interface FormatsDraft {
  conditions: FormatsCondition[];
  /** The saved Custom Format the draft replaces in the test, whose app the draft's values use; omit it to test the draft as a new SlipStream format */
  formatId?: string;
  includeWhenRenaming?: boolean;
  /**
   * @minLength 1
   * @maxLength 100
   */
  name: string;
}
