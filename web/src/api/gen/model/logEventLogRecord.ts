import type { LogEventLogRecordEvent } from "./logEventLogRecordEvent.ts";
import type { SystemLogRecord } from "./systemLogRecord.ts";

export interface LogEventLogRecord {
  event: LogEventLogRecordEvent;
  data: SystemLogRecord;
}
