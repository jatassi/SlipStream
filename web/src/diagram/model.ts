import type { Point3 } from "@anatomy/iso-kit";

import type { NodesTraffic } from "../api/gen/model";
import type { Work } from "../live/activity";
import type { TrafficPair } from "../live/traffic";
import type { Landing, NodeRow } from "../routes/_app/nodes/-landing/view";

// The one pure model of the Diagram's motion: from the landing and the last
// two traffic samples to every rate, speed and fill the live loop eases
// toward, interpolating from the earlier sample to the later over the time
// between them.

/** A polyline measured along its length. */
export type Track = {
  readonly points: readonly Point3[];
  readonly at: readonly number[];
  readonly length: number;
};

export function trackOf(points: readonly Point3[]): Track {
  const at = [0];
  let length = 0;
  for (let index = 1; index < points.length; index++) {
    const a = points[index - 1];
    const b = points[index];
    if (a !== undefined && b !== undefined) {
      length += Math.hypot(b[0] - a[0], b[1] - a[1], b[2] - a[2]);
    }
    at.push(length);
  }
  return { points, at, length };
}

/** The point `s` along a track, clamped to its ends. */
export function pointOn(track: Track, s: number): Point3 {
  const along = Math.min(Math.max(s, 0), track.length);
  let low = 0;
  let high = track.at.length - 1;
  while (high - low > 1) {
    const middle = (low + high) >> 1;
    if ((track.at[middle] ?? 0) <= along) {
      low = middle;
    } else {
      high = middle;
    }
  }
  const a = track.points[low] ?? [0, 0, 0];
  const b = track.points[high] ?? a;
  const from = track.at[low] ?? 0;
  const span = (track.at[high] ?? from) - from;
  const f = span > 0 ? (along - from) / span : 0;
  return [a[0] + (b[0] - a[0]) * f, a[1] + (b[1] - a[1]) * f, a[2] + (b[2] - a[2]) * f];
}

const clamp = (value: number, low: number, high: number): number =>
  Math.min(high, Math.max(low, value));

/** How a link's capsules move: a rate in capsules a second, a speed in world units a second, and the share bound for the Worker. */
export type Flow = {
  readonly worker: string;
  readonly rate: number;
  readonly speed: number;
  readonly towardWorker: number;
};

/** The fewest capsules or pieces a second the Diagram shows while any bytes move. */
export const TRICKLE = 0.3;

/**
 * Capsules a second for a link carrying `bps` bytes a second with a round
 * trip of `rttMs`: none on a quiet link, at least a trickle while bytes
 * move, and a round trip under a millisecond counts as one.
 */
export function capsuleRate(bps: number, rttMs: number): number {
  if (bps <= 0) {
    return 0;
  }
  const mbps = (bps * 8) / 1e6;
  return Math.max(TRICKLE, Math.min(mbps, 0.5 / (Math.max(rttMs, 1) / 1000)) * 0.25);
}

/** How fast a capsule travels for a round trip of `rttMs`: quicker links, quicker capsules. */
export const capsuleSpeed = (rttMs: number): number => 40 + 1900 / Math.max(rttMs, 1);

/** A Node's piece board: the fill of each panel, the pieces a second its antenna takes in, and whether it runs. */
export type Board = {
  readonly node: string;
  readonly panels: readonly number[];
  readonly pieces: number;
};

/**
 * A Node's small machines that follow work: whether its radar dish sweeps,
 * and whether its card catalog's drawer is out.
 */
export type Busy = { readonly node: string; readonly sweep: number; readonly drawer: number };

/** Seconds since each kind of work last reported; infinite when it hasn't. */
export type Ages = Readonly<Record<Work, number>>;

export const QUIET: Ages = { search: Number.POSITIVE_INFINITY, fetch: Number.POSITIVE_INFINITY };

/** How long a dish keeps sweeping after a search last reported: a search reports as it starts and as it finishes. */
export const SWEEP_HOLD = 4;
/** How long a drawer stays out after a fetch. */
export const DRAWER_HOLD = 1.2;

export type Targets = {
  readonly flows: readonly Flow[];
  readonly boards: readonly Board[];
  readonly busy: readonly Busy[];
  /** Each Node's machines run at 1, and wind down to 0 while it is away. */
  readonly running: ReadonlyMap<string, number>;
};

/** The two panels on a board, one per torrent. */
export const PANELS = 2;

/** How far from the earlier sample toward the later a reading `since` ms after the later one lands. */
export function shareOf([before, latest]: TrafficPair, since: number): number {
  if (before === undefined || latest === undefined) {
    return 1;
  }
  const gap = Date.parse(latest.at) - Date.parse(before.at);
  return gap > 0 ? clamp(since / gap, 0, 1) : 1;
}

const mix = (a: number | undefined, b: number | undefined, share: number): number =>
  a === undefined ? (b ?? 0) : a + ((b ?? a) - a) * share;

function flowsOf(landing: Landing, [before, latest]: TrafficPair, share: number): Flow[] {
  return landing.links.map(({ worker, state }) => {
    const now = latest?.links.find(({ b }) => b === worker);
    const was = before?.links.find(({ b }) => b === worker);
    if (state !== "up" || now === undefined) {
      return { worker, rate: 0, speed: capsuleSpeed(60), towardWorker: 0.5 };
    }
    const ab = mix(was?.abBps, now.abBps, share);
    const ba = mix(was?.baBps, now.baBps, share);
    const rtt = mix(was?.rttMs, now.rttMs, share);
    return {
      worker,
      rate: capsuleRate(ab + ba, rtt),
      speed: capsuleSpeed(rtt),
      towardWorker: ab + ba > 0 ? ab / (ab + ba) : 0.5,
    };
  });
}

const doneOf = (
  sample: NodesTraffic["nodes"][number] | undefined,
  grabId: string,
): number | undefined => sample?.torrents.find((torrent) => torrent.grabId === grabId)?.done;

function boardOf(node: NodeRow, samples: TrafficPair, share: number): Board {
  const [before, latest] = samples;
  const now = latest?.nodes.find(({ id }) => id === node.id);
  const was = before?.nodes.find(({ id }) => id === node.id);
  const panels = node.torrents.slice(0, PANELS).map((torrent) => {
    const done = mix(
      doneOf(was, torrent.grabId),
      doneOf(now, torrent.grabId) ?? torrent.done,
      share,
    );
    return torrent.wanted > 0 ? clamp(done / torrent.wanted, 0, 1) : 0;
  });
  const down = mix(was?.downBps, now?.downBps, share);
  return { node: node.id, panels, pieces: down > 0 ? clamp(down / 3e6, TRICKLE, 4) : 0 };
}

const holds = (node: NodeRow, role: NodeRow["roles"][number]["role"]): boolean =>
  node.roles.some((chip) => chip.role === role);

/**
 * Searches and fetches name no Node, so every Node holding Searching sweeps
 * while a search runs, and every Node holding Metadata opens its drawer on
 * a fetch.
 */
const busyOf = (landing: Landing, ages: Ages): Busy[] =>
  landing.nodes
    .filter((node) => holds(node, "searching") || holds(node, "metadata"))
    .map((node) => ({
      node: node.id,
      sweep: holds(node, "searching") && ages.search < SWEEP_HOLD ? 1 : 0,
      drawer: holds(node, "metadata") && ages.fetch < DRAWER_HOLD ? 1 : 0,
    }));

/** Every rate, speed and fill the Diagram shows, `since` ms after the latest sample arrived. */
export function targetsOf(
  landing: Landing,
  samples: TrafficPair,
  since: number,
  ages: Ages,
): Targets {
  const share = shareOf(samples, since);
  return {
    busy: busyOf(landing, ages),
    flows: flowsOf(landing, samples, share),
    boards: landing.nodes
      .filter((node) => node.roles.some(({ role }) => role === "downloading"))
      .map((node) => boardOf(node, samples, share)),
    running: new Map(landing.nodes.map((node) => [node.id, node.state === "online" ? 1 : 0])),
  };
}

/**
 * Where a link's capsules sit in the still frame: evenly spaced, one
 * capsule's travel apart at its rate, inside the ends' fades.
 */
export function stillCapsules(
  length: number,
  { rate, speed }: Flow,
  pool: number,
  margin = 12,
): number[] {
  if (rate < 0.05) {
    return [];
  }
  const spacing = speed / rate;
  const out: number[] = [];
  for (let s = margin; s <= length - margin && out.length < pool; s += spacing) {
    out.push(s);
  }
  return out;
}

/** How many of a panel's cells are lit, whole and in part, for a fill. */
export function litCells(
  fill: number,
  cells: number,
): { readonly whole: number; readonly part: number } {
  const lit = clamp(fill, 0, 1) * cells;
  return { whole: Math.floor(lit), part: lit - Math.floor(lit) };
}

/** A smooth step from 0 at `a` to 1 at `b`. */
export function smooth(a: number, b: number, x: number): number {
  const t = clamp((x - a) / (b - a), 0, 1);
  return t * t * (3 - 2 * t);
}
