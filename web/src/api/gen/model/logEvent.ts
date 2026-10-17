import type { LogEventLogRecord } from "./logEventLogRecord.ts";
import type { LogEventLogReset } from "./logEventLogReset.ts";

export type LogEvent = LogEventLogRecord | LogEventLogReset;
