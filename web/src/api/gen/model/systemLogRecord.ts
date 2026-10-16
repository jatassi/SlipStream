import type { SystemLogRecordLevel } from "./systemLogRecordLevel.ts";

export interface SystemLogRecord {
  /** The record's other attributes */
  attrs: unknown;
  level: SystemLogRecordLevel;
  msg: string;
  seq: number;
  subsystem: string;
  time: string;
}
