import { type Projection, translateAlong } from "@anatomy/iso-kit";
import { follow } from "@anatomy/react/live";

import { type DrawnTile, tileP } from "./drawing";
import { machineP } from "./machines";
import type { Busy } from "./model";
import { DRAWER_TRAVEL } from "./parts/catalog";
import { type HeadPath, REST_HEADING, headOf, paintOf } from "./parts/radar-head";
import { write } from "./svg";

// A tile's small machines that follow work: the radar dish turning while a
// search runs, and the card catalog's drawer sliding out on a fetch.

/** How fast a sweeping dish turns, in degrees a second. */
const SWEEP_SPEED = 60;

/** The step the head turns in, in degrees: each pose is drawn once and kept. */
const POSE_STEP = 2;

const poseOf = (heading: number): number =>
  (((Math.round(heading / POSE_STEP) * POSE_STEP) % 360) + 360) % 360;

/** How many poses one background task draws ahead of the dish. */
const POSES_PER_TASK = 4;

type Poses = Map<number, readonly HeadPath[]>;

// A pose is drawn about the machine's own origin, so every dish seen at one
// scale and angle shares its poses, across tiles and redrawn Diagrams.
const posesByView = new Map<string, Poses>();

const posesOf = ({ scale, azimuth, elevation }: Projection): Poses => {
  const view = `${String(scale)}/${String(azimuth)}/${String(elevation)}`;
  const poses = posesByView.get(view) ?? new Map<number, readonly HeadPath[]>();
  posesByView.set(view, poses);
  return poses;
};

type Head = {
  readonly P: Projection;
  readonly paths: readonly SVGPathElement[];
  readonly poses: Poses;
};

export type MachinesLive = {
  readonly node: string;
  readonly head: Head | undefined;
  readonly led: Element | null;
  readonly drawer: Element | null;
  readonly drawerP: Projection | undefined;
  readonly written: WeakMap<Element, Map<string, string>>;
  /** The background task drawing the poses the dish hasn't turned to yet. */
  ahead: ReturnType<typeof setTimeout> | undefined;
  heading: number;
  drawn: number;
  speed: number;
  out: number;
};

export function machinesOf(
  svg: SVGSVGElement,
  { node, placement }: DrawnTile,
  stage: Projection,
): MachinesLive | undefined {
  const tile = svg.querySelector(`[data-node="${CSS.escape(node.id)}"]`);
  const dish = tile?.querySelector("[data-dish]") ?? null;
  const drawer = tile?.querySelector("[data-drawer-slide]") ?? null;
  if (tile === null || (dish === null && drawer === null)) {
    return undefined;
  }
  const P = tileP(stage, placement);
  const headP = machineP(P, "searching");
  const live: MachinesLive = {
    node: node.id,
    head:
      dish === null
        ? undefined
        : {
            P: headP,
            paths: [...dish.querySelectorAll("path")],
            poses: posesOf(headP),
          },
    led: tile.querySelector('[data-led="search"]'),
    drawer,
    drawerP: drawer === null ? undefined : machineP(P, "metadata"),
    written: new WeakMap(),
    ahead: undefined,
    heading: REST_HEADING,
    drawn: poseOf(REST_HEADING),
    speed: 0,
    out: 0,
  };
  drawAhead(live);
  return live;
}

const poseAt = (head: Head, pose: number): readonly HeadPath[] => {
  const paths = head.poses.get(pose) ?? headOf(head.P, pose);
  head.poses.set(pose, paths);
  return paths;
};

/**
 * Draws every pose between frames, a few at a time, so a turning dish only
 * rewrites its paths: drawing a pose inside a frame costs most of its budget.
 */
function drawAhead(live: MachinesLive) {
  const { head } = live;
  if (head === undefined) {
    return;
  }
  const missing = Array.from({ length: 360 / POSE_STEP }, (_, index) => index * POSE_STEP).filter(
    (pose) => !head.poses.has(pose),
  );
  const next = () => {
    for (const pose of missing.splice(0, POSES_PER_TASK)) {
      poseAt(head, pose);
    }
    live.ahead = missing.length === 0 ? undefined : setTimeout(next, 0);
  };
  live.ahead = missing.length === 0 ? undefined : setTimeout(next, 0);
}

export function disposeMachines(live: MachinesLive) {
  clearTimeout(live.ahead);
}

function drawHead(live: MachinesLive, heading: number) {
  const { head } = live;
  if (head === undefined) {
    return;
  }
  const pose = poseOf(heading);
  poseAt(head, pose).forEach(({ paint, d }, index) => {
    const path = head.paths[index] ?? null;
    write(live.written, path, "d", d);
    for (const [name, value] of Object.entries(paintOf(paint))) {
      write(live.written, path, name, value);
    }
  });
  live.drawn = pose;
}

function slide(live: MachinesLive, out: number) {
  if (live.drawerP !== undefined) {
    write(
      live.written,
      live.drawer,
      "transform",
      out < 0.002 ? "" : translateAlong(live.drawerP, [0, out * DRAWER_TRAVEL, 0]),
    );
  }
}

/** One frame: the dish eases to its sweep and turns on, the drawer eases out or home. */
export function moveMachines(live: MachinesLive, busy: Busy, running: number, dt: number) {
  live.speed = follow(live.speed, busy.sweep * running * SWEEP_SPEED, dt, 0.5);
  live.heading = (live.heading + live.speed * dt) % 360;
  if (poseOf(live.heading) !== live.drawn) {
    drawHead(live, live.heading);
  }
  write(live.written, live.led, "opacity", (0.2 + 0.8 * (live.speed / SWEEP_SPEED)).toFixed(2));
  live.out = follow(live.out, busy.drawer * running, dt, 0.16);
  slide(live, live.out);
}

/** The still frame: the dish at rest facing its way, the drawer home, the lamp out. */
export function stillMachines(live: MachinesLive) {
  live.speed = 0;
  live.heading = REST_HEADING;
  drawHead(live, REST_HEADING);
  write(live.written, live.led, "opacity", "0.2");
  live.out = 0;
  slide(live, 0);
}
