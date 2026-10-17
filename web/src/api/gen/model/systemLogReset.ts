import type { SystemLogResetReason } from "./systemLogResetReason.ts";

export interface SystemLogReset {
  reason: SystemLogResetReason;
  /** When the serving process started */
  startedAt: string;
}
