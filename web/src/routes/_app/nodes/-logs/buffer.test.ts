import { expect, test } from "vitest";

import {
  type LogEvent,
  type LogRecord,
  type LogRow,
  type LogView,
  createLogBuffer,
  emptyView,
  lineOf,
  logReducer,
  rowKey,
} from "./buffer";

const record = (seq: number, over: Partial<LogRecord> = {}): LogRecord => ({
  seq,
  time: "2026-10-16T09:00:00Z",
  level: "info",
  subsystem: "http",
  msg: `line ${seq}`,
  attrs: {},
  ...over,
});

const records = (from: number, to: number): LogEvent => ({
  type: "records",
  records: Array.from({ length: to - from + 1 }, (_, i) => record(from + i)),
});

const run = (cap: number, ...events: LogEvent[]): LogView =>
  events.reduce((view, event) => logReducer(view, event), emptyView(cap));

const seqs = (rows: readonly LogRow[]) => rows.map((row) => (row.kind === "line" ? row.seq : "|"));

test("a line's haystack is its message, subsystem and attribute values, lowercased", () => {
  const row = lineOf(
    record(1, {
      msg: "Grabbed Release",
      subsystem: "Search",
      attrs: { id: 42, who: { n: "Ada" } },
    }),
  );
  expect(row).toMatchObject({ kind: "line", seq: 1 });
  expect(row.kind === "line" && row.haystack).toBe('grabbed release\nsearch\n42\n{"n":"ada"}');
});

test("the buffer keeps the newest lines up to its cap", () => {
  const view = run(3, records(1, 2), records(3, 5));
  expect(seqs(view.lines)).toEqual([3, 4, 5]);
  expect(view.waiting).toEqual([]);
});

test("a reset leaves a divider between the lines before and after it", () => {
  const view = run(
    10,
    records(1, 2),
    { type: "reset", reason: "restarted", at: "t" },
    records(3, 3),
  );
  expect(seqs(view.lines)).toEqual([1, 2, "|", 3]);
  expect(view.lines[2]).toEqual({ kind: "divider", reason: "restarted", at: "t" });
  expect(view.lines.map(rowKey)).toEqual(["line-1", "line-2", "restarted-t", "line-3"]);
});

test("pause holds the shown lines and counts what arrives, and resume appends it", () => {
  const paused = run(5, records(1, 4), { type: "pause" }, records(5, 7));
  expect(seqs(paused.lines)).toEqual([1, 2, 3, 4]);
  expect(seqs(paused.waiting)).toEqual([5, 6, 7]);
  expect(paused.dropped).toBe(false);
  const resumed = logReducer(paused, { type: "resume" });
  expect(seqs(resumed.lines)).toEqual([3, 4, 5, 6, 7]);
  expect(resumed).toMatchObject({ waiting: [], paused: false, dropped: false });
});

test("a long pause keeps the newest arrivals up to the cap and says the oldest dropped", () => {
  const paused = run(3, records(1, 3), { type: "pause" }, records(4, 6), records(7, 8));
  expect(seqs(paused.lines)).toEqual([1, 2, 3]);
  expect(seqs(paused.waiting)).toEqual([6, 7, 8]);
  expect(paused.dropped).toBe(true);
  expect(seqs(logReducer(paused, { type: "resume" }).lines)).toEqual([6, 7, 8]);
});

test("clear empties the view and keeps it paused", () => {
  const view = run(5, records(1, 2), { type: "pause" }, records(3, 3), { type: "clear" });
  expect(view).toEqual({ ...emptyView(5), paused: true });
});

test("resizing trims to the newest lines", () => {
  const view = run(5, records(1, 5), { type: "pause" }, records(6, 9), { type: "resize", cap: 2 });
  expect(view.cap).toBe(2);
  expect(seqs(view.lines)).toEqual([4, 5]);
  expect(seqs(view.waiting)).toEqual([8, 9]);
  expect(view.dropped).toBe(true);
});

test("the buffer notifies only when the view changes", () => {
  const buffer = createLogBuffer(5);
  let heard = 0;
  buffer.subscribe(() => {
    heard += 1;
  });
  buffer.dispatch({ type: "resume" });
  buffer.dispatch({ type: "records", records: [] });
  buffer.dispatch(records(1, 1));
  buffer.dispatch({ type: "resize", cap: 5 });
  expect(heard).toBe(1);
  expect(seqs(buffer.getSnapshot().lines)).toEqual([1]);
});
