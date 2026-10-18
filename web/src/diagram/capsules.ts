import { type Projection, iso } from "@anatomy/iso-kit";
import { follow } from "@anatomy/react/live";

import type { DrawnLink } from "./drawing";
import { type Flow, type Track, pointOn, smooth, stillCapsules, trackOf } from "./model";
import { fixed, hide, make } from "./svg";

// The capsules that carry a link's traffic: a pool per link, spawned at the
// link's rate in each direction and moved at its speed, glowing brightest at
// the end that dialed.

const POOL = 28;
const HALF = 3.2;

type Capsule = {
  s: number;
  toWorker: boolean;
  shown: boolean;
  readonly g: SVGGElement;
  readonly paths: readonly SVGPathElement[];
};

export type Capsules = {
  readonly worker: string;
  readonly track: Track;
  readonly dialerAtWorker: boolean | undefined;
  readonly pool: readonly Capsule[];
  toWorker: number;
  toController: number;
  rate: number;
  speed: number;
};

export function capsulesOf(
  svg: SVGSVGElement,
  { route, dialerAtWorker }: DrawnLink,
  k: number,
): Capsules | undefined {
  const layer = svg.querySelector(`[data-link="${route.worker}"] [data-capsules]`);
  if (layer === null) {
    return undefined;
  }
  const pool = Array.from({ length: POOL }, (): Capsule => {
    const g = make("g", { opacity: "0" }, layer);
    const halo = make("path", { class: "capsule capsule-halo", "stroke-width": fixed(5.2 * k) }, g);
    const core = make("path", { class: "capsule capsule-core", "stroke-width": fixed(2 * k) }, g);
    return { s: -1, toWorker: true, shown: false, g, paths: [halo, core] };
  });
  return {
    worker: route.worker,
    track: trackOf(route.points),
    dialerAtWorker,
    pool,
    toWorker: 0,
    toController: 0,
    rate: 0,
    speed: 0,
  };
}

function place(link: Capsules, capsule: Capsule, P: Projection) {
  const { track } = link;
  const along = capsule.toWorker ? capsule.s : track.length - capsule.s;
  const a = iso(pointOn(track, along - HALF), P);
  const b = iso(pointOn(track, along + HALF), P);
  const d = `M${fixed(a[0])} ${fixed(a[1])}L${fixed(b[0])} ${fixed(b[1])}`;
  for (const path of capsule.paths) {
    path.setAttribute("d", d);
  }
  const fromDialer =
    link.dialerAtWorker === undefined
      ? track.length / 2
      : link.dialerAtWorker
        ? track.length - along
        : along;
  const glow = 0.45 + 0.55 * (1 - fromDialer / track.length);
  const fade = smooth(0, 8, along) * smooth(0, 8, track.length - along);
  capsule.g.setAttribute("opacity", (fade * glow).toFixed(3));
  capsule.shown = true;
}

function spawn(link: Capsules, toWorker: boolean) {
  const free = link.pool.find((capsule) => capsule.s < 0);
  if (free !== undefined) {
    free.s = 0;
    free.toWorker = toWorker;
  }
}

/** One frame of a link: ease to its flow, spawn what its rate owes, move the rest on. */
export function moveCapsules(
  link: Capsules,
  flow: Flow,
  running: number,
  dt: number,
  P: Projection,
) {
  link.rate = follow(link.rate, flow.rate * running, dt, 0.6);
  link.speed = follow(link.speed, flow.speed, dt, 0.6);
  link.toWorker += link.rate * flow.towardWorker * dt;
  link.toController += link.rate * (1 - flow.towardWorker) * dt;
  for (; link.toWorker >= 1; link.toWorker -= 1) {
    spawn(link, true);
  }
  for (; link.toController >= 1; link.toController -= 1) {
    spawn(link, false);
  }
  for (const capsule of link.pool) {
    if (capsule.s >= 0) {
      capsule.s += link.speed * dt;
    }
    if (capsule.s < 0 || capsule.s > link.track.length) {
      capsule.s = -1;
      hide(capsule, capsule.g);
    } else {
      place(link, capsule, P);
    }
  }
}

/** The still frame of a link: capsules evenly spaced at its flow. */
export function stillCapsulesOn(link: Capsules, flow: Flow | undefined, P: Projection) {
  const placed = flow === undefined ? [] : stillCapsules(link.track.length, flow, POOL);
  link.pool.forEach((capsule, index) => {
    const s = placed[index];
    capsule.s = s ?? -1;
    capsule.toWorker = true;
    if (s === undefined) {
      hide(capsule, capsule.g);
    } else {
      place(link, capsule, P);
    }
  });
}

export const disposeCapsules = (link: Capsules) => {
  for (const capsule of link.pool) {
    capsule.g.remove();
  }
};
