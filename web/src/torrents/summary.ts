import type { DownloadclientsCounts } from "../api/gen/model";
import { type Clock, sinceWords } from "../format";
import type { StatusPart } from "../ui/status-sentence";
import type { Row, State } from "./model";

export type Signal = {
  readonly anchor: string;
  readonly signal: "fail" | "warn";
  readonly chip: string;
  readonly verdict: string;
  readonly rows: readonly Row[];
};

type Cause = Omit<Signal, "anchor" | "rows"> & { readonly key: string };

function causeOf(row: Row, node: string, clock: Clock): Cause | undefined {
  const host =
    row.tracker === undefined || row.tracker.host === "" ? "Its announce URL" : row.tracker.host;
  switch (row.severity) {
    case "none":
      return undefined;
    case "failure":
      return row.state === "failed"
        ? {
            key: `failed:${row.error ?? ""}`,
            signal: "fail",
            chip: "Failed",
            verdict:
              row.error === undefined
                ? "The torrent client stopped it."
                : `The torrent client stopped it: ${row.error}.`,
          }
        : {
            key: `unregistered:${host}`,
            signal: "fail",
            chip: "Unregistered",
            verdict: `${host} reports that the torrent is no longer registered.`,
          };
    case "warning":
      return warningOf(row, host, node, clock);
    default:
      return row.severity satisfies never;
  }
}

function warningOf(row: Row, host: string, node: string, clock: Clock): Cause {
  if (row.state === "stalled") {
    return {
      key: "stalled",
      signal: "warn",
      chip: "Stalled",
      verdict: "No data has arrived for a while, so it no longer holds a download slot.",
    };
  }
  if (row.hold === "disk_full") {
    return {
      key: `disk_full:${row.nodeId}`,
      signal: "warn",
      chip: "Disk full",
      verdict: `${node} ran out of disk space. Downloads resume once there is space.`,
    };
  }
  if (row.hold === "disk_guard") {
    return {
      key: `disk_guard:${row.nodeId}`,
      signal: "warn",
      chip: "Waiting for disk",
      verdict: `New downloads on ${node} wait for disk space.`,
    };
  }
  const said = row.tracker?.message;
  const retry = row.tracker?.retryAt;
  return {
    key: `tracker:${host}:${said ?? ""}`,
    signal: "warn",
    chip: "Announce error",
    verdict: `${said === undefined ? `${host} isn't responding.` : `${host} reported "${said}".`}${
      retry === undefined ? "" : ` The client retries at ${sinceWords(retry, clock)}.`
    }`,
  };
}

/**
 * Failures, then warnings, each cause once with every Torrent it holds, the
 * cause with the oldest Torrent first. A tracker that goes down for 500
 * seeds is one signal, not 500.
 */
const oldest = (group: readonly Row[]) => Math.min(...group.map((row) => Date.parse(row.addedAt)));

export function signalsOf(
  rows: readonly Row[],
  nodeName: (id: string) => string,
  clock: Clock,
): Signal[] {
  const causes = new Map<string, { cause: Cause; rows: Row[] }>();
  for (const row of rows) {
    const cause = causeOf(row, nodeName(row.nodeId), clock);
    if (cause !== undefined) {
      const known = causes.get(cause.key);
      if (known === undefined) {
        causes.set(cause.key, { cause, rows: [row] });
      } else {
        known.rows.push(row);
      }
    }
  }
  const rank = { fail: 0, warn: 1 } as const;
  return [...causes.values()]
    .toSorted(
      (a, b) => rank[a.cause.signal] - rank[b.cause.signal] || oldest(a.rows) - oldest(b.rows),
    )
    .map(({ cause, rows: held }, index) => ({
      anchor: `torrent-signal-${index}`,
      signal: cause.signal,
      chip: cause.chip,
      verdict: cause.verdict,
      rows: held,
    }));
}

const countOf = (counts: DownloadclientsCounts, state: State) =>
  counts.byState.find((count) => count.state === state)?.torrents ?? 0;

const plural = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`;

export function torrentsSentence(
  counts: DownloadclientsCounts,
  signals: readonly Signal[],
): StatusPart[] {
  const downloading = countOf(counts, "downloading");
  const seeding = countOf(counts, "seeding");
  const parts: StatusPart[] = [
    downloading === 0
      ? { kind: "plain", text: "Nothing downloading" }
      : { kind: "live", text: `${downloading} downloading` },
  ];
  if (seeding > 0) {
    parts.push({ kind: "plain", text: `${seeding} seeding` });
  }
  for (const [signal, kind, one, many] of [
    ["fail", "failure", "failure", "failures"],
    ["warn", "warning", "warning", "warnings"],
  ] as const) {
    const held = signals.filter((each) => each.signal === signal);
    const [first] = held;
    if (first !== undefined) {
      const torrents = held.reduce((sum, each) => sum + each.rows.length, 0);
      parts.push({ kind, text: plural(torrents, one, many), href: `#${first.anchor}` });
    }
  }
  return parts;
}
