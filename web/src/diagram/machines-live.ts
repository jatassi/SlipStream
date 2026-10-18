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

type Head = {
  readonly P: Projection;
  readonly paths: readonly SVGPathElement[];
  readonly poses: Map<number, readonly HeadPath[]>;
};

export type MachinesLive = {
  readonly node: string;
  readonly head: Head | undefined;
  readonly led: Element | null;
  readonly drawer: Element | null;
  readonly drawerP: Projection | undefined;
  readonly written: WeakMap<Element, Map<string, string>>;
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
  return {
    node: node.id,
    head:
      dish === null
        ? undefined
        : {
            P: machineP(P, "searching"),
            paths: [...dish.querySelectorAll("path")],
            poses: new Map(),
          },
    led: tile.querySelector('[data-led="search"]'),
    drawer,
    drawerP: drawer === null ? undefined : machineP(P, "metadata"),
    written: new WeakMap(),
    heading: REST_HEADING,
    drawn: poseOf(REST_HEADING),
    speed: 0,
    out: 0,
  };
}

function drawHead(live: MachinesLive, heading: number) {
  const { head } = live;
  if (head === undefined) {
    return;
  }
  const pose = poseOf(heading);
  const paths = head.poses.get(pose) ?? headOf(head.P, pose);
  head.poses.set(pose, paths);
  paths.forEach(({ paint, d }, index) => {
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
