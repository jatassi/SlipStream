import type {
  DownloadclientsCounts,
  DownloadclientsTabCountTab,
  DownloadclientsTorrentRow,
  DownloadclientsTorrentRowState,
  DownloadclientsTrackerCell,
} from "../api/gen/model";
import { type Clock, bytes, lengthWords, sinceWords } from "../format";
import type { LiveRow } from "../live/torrents";

export type Row = DownloadclientsTorrentRow;
export type State = DownloadclientsTorrentRowState;
export type Tab = DownloadclientsTabCountTab;

export const tabLabels: { readonly [T in Tab]: string } = {
  downloading: "Downloading",
  seeding: "Seeding",
  problems: "Problems",
  all: "All",
};

export const tabOrder = {
  desktop: ["downloading", "seeding", "problems", "all"],
  phone: ["all", "downloading", "seeding", "problems"],
} as const satisfies Record<string, readonly Tab[]>;

export const stateWords: { readonly [S in State]: string } = {
  removing: "Removing",
  failed: "Failed",
  "tracker-error": "Tracker error",
  paused: "Paused",
  checking: "Checking",
  queued: "Queued",
  stalled: "Stalled",
  downloading: "Downloading",
  seeding: "Seeding",
};

export type TorrentsFilter = {
  readonly tab: Tab;
  readonly node?: string | undefined;
  readonly type?: string | undefined;
  readonly q: string;
};

export function matches(row: Row, filter: TorrentsFilter, work: string | undefined): boolean {
  const q = filter.q.trim().toLowerCase();
  return (
    row.tabs.includes(filter.tab) &&
    (filter.node === undefined || row.nodeId === filter.node) &&
    (filter.type === undefined || row.contentType === filter.type) &&
    (q === "" || row.name.toLowerCase().includes(q) || (work?.toLowerCase().includes(q) ?? false))
  );
}

export type Place = { readonly node: string; readonly clock: Clock };

const ordinalSuffix: Readonly<Record<Intl.LDMLPluralRule, string>> = {
  zero: "th",
  one: "st",
  two: "nd",
  few: "rd",
  many: "th",
  other: "th",
};
const ordinals = new Intl.PluralRules("en", { type: "ordinal" });
export const ordinal = (n: number) => `${n}${ordinalSuffix[ordinals.select(n)]}`;

const peerWords = (live: LiveRow | undefined) =>
  live === undefined ? "" : `${live.peers} ${live.peers === 1 ? "peer" : "peers"}`;

function pendingLine(row: Row, { node }: Place): string | undefined {
  const away = row.lastKnown !== undefined;
  switch (row.pending) {
    case "pause":
      return away ? `Pauses when ${node} is back` : "Pausing";
    case "resume":
      return away ? `Resumes when ${node} is back` : "Resuming";
    case undefined:
      return undefined;
    default:
      return row.pending satisfies never;
  }
}

function heldLine(row: Row, { node, clock }: Place): string {
  switch (row.hold) {
    case "window":
      return row.until === undefined
        ? "Waiting for its tracker"
        : `Resumes at ${sinceWords(row.until, clock)}`;
    case "disk_full":
      return "Disk full, resumes once there is space";
    case "slot":
      return row.queuePosition === undefined
        ? "Waiting for a slot"
        : `${ordinal(row.queuePosition)} in line`;
    case "check_slot":
      return "Waiting to check";
    case "disk_guard":
      return `Waiting for disk space on ${node}`;
    case "draining":
      return `Waiting because ${node} is draining`;
    case "controller":
      return "Waiting for the Controller";
    case "wanted":
    case undefined:
      return "";
    default:
      return row.hold satisfies never;
  }
}

export function stateLine(row: Row, live: LiveRow | undefined, place: Place): string {
  const pending = pendingLine(row, place);
  if (pending !== undefined) {
    return pending;
  }
  if (row.state === "removing") {
    return `Waiting for ${place.node} to confirm`;
  }
  if (row.lastKnown !== undefined) {
    return `Last known at ${sinceWords(row.lastKnown, place.clock)}`;
  }
  switch (row.state) {
    case "failed":
      return row.error ?? "The torrent client stopped it";
    case "tracker-error":
      return trackerLine(row.tracker, place.clock).status;
    case "checking":
      return row.until === undefined
        ? "Checking its files"
        : `Rechecks, then resumes at ${sinceWords(row.until, place.clock)}`;
    case "queued":
      return row.hold === undefined ? `Waiting for ${place.node}` : heldLine(row, place);
    case "paused":
      return heldLine(row, place);
    case "stalled":
      return "No data is arriving";
    case "downloading":
    case "seeding":
      return peerWords(live);
    default:
      return row.state satisfies never;
  }
}

export function trackerLine(
  cell: DownloadclientsTrackerCell | undefined,
  clock: Clock,
): { host: string; status: string } {
  if (cell === undefined) {
    return { host: "No tracker", status: "" };
  }
  const host = cell.host === "" ? "Its tracker" : cell.host;
  const retry =
    cell.retryAt === undefined ? "" : `, retrying at ${sinceWords(cell.retryAt, clock)}`;
  switch (cell.status) {
    case "working":
      return { host, status: "Working" };
    case "updating":
      return { host, status: "Announcing" };
    case "not_contacted":
      return { host, status: "Not contacted yet" };
    case "unregistered":
      return { host, status: `Unregistered by ${host}` };
    case "error":
      return {
        host,
        status: `${cell.message === undefined ? "Not responding" : `Error: ${cell.message}`}${retry}`,
      };
    default:
      return cell.status satisfies never;
  }
}

export function progressOf(row: Row, live: LiveRow | undefined): { done: number; wanted: number } {
  return live === undefined
    ? { done: row.done, wanted: row.size }
    : { done: live.done, wanted: live.wanted };
}

export const percentOf = ({ done, wanted }: { done: number; wanted: number }): number =>
  wanted === 0 ? 100 : Math.floor((done / wanted) * 100);

export function rateWords(bps: number): string {
  if (bps <= 0) {
    return "—";
  }
  return bps < 1024 ? `${(bps / 1024).toFixed(1)} KB/s` : `${bytes(bps)}/s`;
}

/** Time left, in whole minutes rounded up: "2 min left". */
export const leftWords = (seconds: number): string =>
  `${lengthWords(Math.ceil(seconds / 60) * 60_000)} left`;

function etaWords(live: LiveRow): string {
  if (live.downBps <= 0 || live.done >= live.wanted) {
    return "";
  }
  return leftWords((live.wanted - live.done) / live.downBps);
}

/**
 * A Torrent's time left and download speed, or none while it doesn't
 * download: a paused or queued torrent's live rate is the last one winding
 * down, not a pace.
 */
export function paceOf(
  state: State | undefined,
  live: LiveRow | undefined,
): { readonly eta: string; readonly rate: string } {
  if (state !== "downloading" || live === undefined) {
    return { eta: "", rate: "" };
  }
  return { eta: etaWords(live), rate: live.downBps > 0 ? rateWords(live.downBps) : "" };
}

const hour = 3600;
const day = 24 * hour;

export function seedLength(seconds: number): string {
  if (seconds < hour) {
    return `${Math.floor(seconds / 60)} min`;
  }
  if (seconds < 2 * day) {
    return `${Math.floor(seconds / hour)} h`;
  }
  return `${Math.floor(seconds / day)} days`;
}

/** A seed's progress cell, until seeding goals take its place. */
export const seedWords = ({ seedingSeconds }: Row): string =>
  seedingSeconds < 60 ? "Seeding since just now" : `Seeded ${seedLength(seedingSeconds)}`;

export const tabCount = (counts: DownloadclientsCounts, tab: Tab): number =>
  counts.byTab.find((count) => count.tab === tab)?.torrents ?? 0;
