import { expect, test } from "vitest";

import { type LogRecord, type LogRow, lineOf } from "./buffer";
import { isFiltered, noFilter, subsystemsOf, visibleRows } from "./filter";

const line = (seq: number, over: Partial<LogRecord>): LogRow =>
  lineOf({
    seq,
    time: "2026-10-16T09:00:00Z",
    level: "info",
    subsystem: "http",
    msg: "",
    attrs: {},
    ...over,
  });

const rows: readonly LogRow[] = [
  line(1, { level: "debug", subsystem: "search", msg: "Querying indexers" }),
  line(2, { level: "info", subsystem: "http", msg: "Listening", attrs: { port: 8484 } }),
  { kind: "divider", reason: "restarted", at: "2026-10-16T09:01:00Z" },
  line(3, {
    level: "warn",
    subsystem: "Import",
    msg: "Slow disk",
    attrs: { path: "/Media/Movies" },
  }),
  line(4, { level: "error", subsystem: "search", msg: "Indexer down" }),
];

const shown = (filter: Partial<typeof noFilter>) =>
  visibleRows(rows, { ...noFilter, ...filter }).map((row) => (row.kind === "line" ? row.seq : "|"));

test("no filter keeps every row", () => {
  expect(isFiltered(noFilter)).toBe(false);
  expect(visibleRows(rows, noFilter)).toBe(rows);
});

test("the minimum level keeps that level and above, and dividers always stay", () => {
  expect(shown({ min: "warn" })).toEqual(["|", 3, 4]);
  expect(shown({ min: "error" })).toEqual(["|", 4]);
});

test("the subsystem filter keeps one subsystem", () => {
  expect(shown({ subsystem: "search" })).toEqual([1, "|", 4]);
});

test("search matches the message, subsystem and attribute values, ignoring case", () => {
  expect(shown({ search: "INDEXER" })).toEqual([1, "|", 4]);
  expect(shown({ search: "import" })).toEqual(["|", 3]);
  expect(shown({ search: "8484" })).toEqual([2, "|"]);
  expect(shown({ search: "media/movies" })).toEqual(["|", 3]);
  expect(shown({ search: "  " })).toEqual([1, 2, "|", 3, 4]);
});

test("filters combine", () => {
  expect(shown({ min: "info", subsystem: "search", search: "down" })).toEqual(["|", 4]);
});

test("the subsystems are each line's, once, sorted", () => {
  expect(subsystemsOf(rows)).toEqual(["http", "Import", "search"]);
});
