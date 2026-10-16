import type { SettingTimes } from "./settingTimes.ts";

export interface TasksSchedulePatch {
  /**
   * Seconds; null resets it to its default
   * @nullable
   */
  headerDelay?: number | null;
  /**
   * Seconds; null resets it to its default
   * @nullable
   */
  jitterCap?: number | null;
  /**
   * null resets it to its default
   * @nullable
   */
  jitterPercent?: number | null;
  /**
   * catch-up or wait; null resets it to its default
   * @nullable
   */
  overdue?: string | null;
  /**
   * null resets it to its default
   * @nullable
   */
  partialWarnIntervals?: number | null;
  /**
   * null resets it to its default
   * @nullable
   */
  runsKept?: number | null;
  /** null resets it to its default */
  window?: SettingTimes | null;
}
