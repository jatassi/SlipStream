import type { LogEventLogResetEvent } from "./logEventLogResetEvent.ts";
import type { SystemLogReset } from "./systemLogReset.ts";

export interface LogEventLogReset {
  event: LogEventLogResetEvent;
  data: SystemLogReset;
}
