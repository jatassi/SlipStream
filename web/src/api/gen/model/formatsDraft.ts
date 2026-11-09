import type { FormatsCondition } from "./formatsCondition.ts";

export interface FormatsDraft {
  conditions: FormatsCondition[];
  /** The saved Custom Format the draft stands in for, whose app it is written in; left out, the draft is a new format in SlipStream's own values */
  formatId?: string;
  includeWhenRenaming?: boolean;
  /**
   * @minLength 1
   * @maxLength 100
   */
  name: string;
}
