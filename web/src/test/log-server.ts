import { sse } from "msw";

import type { SystemLogRecord, SystemLogReset } from "../api/gen/model";

type LogMap = { "log.record": SystemLogRecord; "log.reset": SystemLogReset };
type Client = Parameters<Parameters<typeof sse<LogMap>>[1]>[0]["client"];

/** A record at `seq`, a millisecond apart from the one before. */
export const logRecord = (seq: number, record: Partial<SystemLogRecord> = {}): SystemLogRecord => ({
  seq,
  time: new Date(Date.UTC(2026, 9, 12, 9, 0, 0, seq)).toISOString(),
  level: "info",
  subsystem: "node",
  msg: `Line ${seq}`,
  attrs: {},
  ...record,
});

/**
 * A Node's log stream: each reader first gets the records of `ring` after
 * the one it resumes from, then what the test sends. `resumes` keeps the
 * sequence number each reader resumed from, 0 for the first.
 */
export function logServer(ring: readonly SystemLogRecord[] = []) {
  const clients: Client[] = [];
  const resumes: number[] = [];
  const handler = sse<LogMap>("*/api/v1/nodes/:id/logs/stream", ({ client, request }) => {
    clients.push(client);
    const after = Number(
      new URL(request.url).searchParams.get("after") ?? request.headers.get("Last-Event-ID") ?? 0,
    );
    resumes.push(after);
    for (const record of ring.filter((one) => one.seq > after)) {
      client.send({ event: "log.record", data: record, id: String(record.seq) });
    }
  });
  const reader = () => {
    const client = clients.at(-1);
    if (client === undefined) {
      throw new Error("no reader has opened the log stream");
    }
    return client;
  };
  return {
    handler,
    clients,
    resumes,
    send(...records: readonly SystemLogRecord[]) {
      for (const record of records) {
        reader().send({ event: "log.record", data: record, id: String(record.seq) });
      }
    },
    reset(reset: SystemLogReset, after: number) {
      reader().send({ event: "log.reset", data: reset, id: String(after) });
    },
  };
}
