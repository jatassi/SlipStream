import type { FormatsCustomFormat } from "./formatsCustomFormat.ts";

export interface FormatsFormatList {
  /** By name */
  formats: FormatsCustomFormat[];
  /**
   * How long one pattern may run against one name before it counts as not matching
   * @minimum 0
   */
  matchLimitMilliseconds: number;
}
