import { createStore } from "../../../../shell/store";

export type Level = "debug" | "info" | "warn" | "error";

export type LogRecord = {
  readonly seq: number;
  readonly time: string;
  readonly level: Level;
  readonly subsystem: string;
  readonly msg: string;
  readonly attrs: Readonly<Record<string, unknown>>;
};

export type ResetReason = "restarted" | "lapped";

export type LogRow =
  | ({ readonly kind: "line"; readonly haystack: string } & LogRecord)
  | { readonly kind: "divider"; readonly reason: ResetReason; readonly at: string };

/**
 * What the Logs section shows. While paused, `lines` holds still and arrivals
 * wait in `waiting`, which keeps the newest `cap` and sets `dropped` past it.
 */
export type LogView = {
  readonly cap: number;
  readonly lines: readonly LogRow[];
  readonly waiting: readonly LogRow[];
  readonly paused: boolean;
  readonly dropped: boolean;
};

export type LogEvent =
  | { readonly type: "records"; readonly records: readonly LogRecord[] }
  | { readonly type: "reset"; readonly reason: ResetReason; readonly at: string }
  | { readonly type: "pause" }
  | { readonly type: "resume" }
  | { readonly type: "clear" }
  | { readonly type: "resize"; readonly cap: number };

const attrText = (value: unknown): string =>
  typeof value === "object" && value !== null ? JSON.stringify(value) : String(value);

export function lineOf(record: LogRecord): LogRow {
  const words = [
    record.msg,
    record.subsystem,
    ...Object.values(record.attrs).map((value) => attrText(value)),
  ];
  return { kind: "line", ...record, haystack: words.join("\n").toLowerCase() };
}

export const rowKey = (row: LogRow): string =>
  row.kind === "line" ? `line-${row.seq}` : `${row.reason}-${row.at}`;

export const emptyView = (cap: number): LogView => ({
  cap,
  lines: [],
  waiting: [],
  paused: false,
  dropped: false,
});

const newest = (rows: readonly LogRow[], cap: number) =>
  rows.length > cap ? rows.slice(rows.length - cap) : rows;

function arrive(view: LogView, rows: readonly LogRow[]): LogView {
  if (rows.length === 0) {
    return view;
  }
  if (!view.paused) {
    return { ...view, lines: newest([...view.lines, ...rows], view.cap) };
  }
  const waiting = [...view.waiting, ...rows];
  return {
    ...view,
    waiting: newest(waiting, view.cap),
    dropped: view.dropped || waiting.length > view.cap,
  };
}

export function logReducer(view: LogView, event: LogEvent): LogView {
  if (event.type === "records") {
    return arrive(
      view,
      event.records.map((one) => lineOf(one)),
    );
  }
  if (event.type === "reset") {
    return arrive(view, [{ kind: "divider", reason: event.reason, at: event.at }]);
  }
  if (event.type === "pause") {
    return view.paused ? view : { ...view, paused: true };
  }
  if (event.type === "resume") {
    if (!view.paused) {
      return view;
    }
    const lines = newest([...view.lines, ...view.waiting], view.cap);
    return { ...view, lines, waiting: [], paused: false, dropped: false };
  }
  if (event.type === "clear") {
    return { ...emptyView(view.cap), paused: view.paused };
  }
  if (event.cap === view.cap) {
    return view;
  }
  return {
    ...view,
    cap: event.cap,
    lines: newest(view.lines, event.cap),
    waiting: newest(view.waiting, event.cap),
    dropped: view.dropped || view.waiting.length > event.cap,
  };
}

export const createLogBuffer = (cap: number) => createStore(logReducer, emptyView(cap));
