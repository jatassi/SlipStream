import type { ConversionsHours } from "./conversionsHours.ts";

export interface ConversionsNodeSettings {
  /**
   * null for the default
   * @minimum 1
   * @maximum 16
   * @nullable
   */
  atOnce: number | null;
  hours: ConversionsHours;
  processor: boolean;
}
