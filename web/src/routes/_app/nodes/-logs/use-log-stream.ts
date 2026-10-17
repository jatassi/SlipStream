import { useEffect, useState, useSyncExternalStore } from "react";
import { z } from "zod";

import { type LogStreamEvent, openLogStream } from "../../../../live/log-stream";
import { type LogEvent, type LogRecord, type LogView, createLogBuffer } from "./buffer";

const attrsOf = z.record(z.string(), z.unknown());

// Records in a row become one buffer event, so a frame's batch is one update.
function bufferEvents(events: readonly LogStreamEvent[]): LogEvent[] {
  const out: LogEvent[] = [];
  let records: LogRecord[] = [];
  const flushRecords = () => {
    if (records.length > 0) {
      out.push({ type: "records", records });
      records = [];
    }
  };
  for (const event of events) {
    if (event.event === "log.record") {
      records.push({ ...event.data, attrs: attrsOf.parse(event.data.attrs) });
    } else {
      flushRecords();
      out.push({
        type: "reset",
        reason: event.data.reason,
        at: event.data.startedAt,
        after: Number(event.id),
      });
    }
  }
  flushRecords();
  return out;
}

/** One Node's log held in the browser: at most `cap` lines, with pause and clear. */
export function useLogStream(nodeId: string, cap: number) {
  const [buffer] = useState(() => createLogBuffer(cap));
  useEffect(() => {
    buffer.dispatch({ type: "resize", cap });
  }, [buffer, cap]);
  useEffect(() => {
    const stream = openLogStream(nodeId, (events) => {
      for (const event of bufferEvents(events)) {
        buffer.dispatch(event);
      }
    });
    return () => {
      stream.close();
    };
  }, [buffer, nodeId]);
  const view: LogView = useSyncExternalStore(buffer.subscribe, buffer.getSnapshot);
  return { view, dispatch: buffer.dispatch };
}
