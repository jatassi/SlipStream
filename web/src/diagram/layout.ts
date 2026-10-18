import type { Plan, Plane, Point3 } from "@anatomy/iso-kit";

/**
 * The canonical tile: the Controller's site, centred on its anchor, standing
 * on the floor at z = 0. A Worker's tile is the same tile at WORKER_SCALE.
 */
export const TILE: Plan = { x: -170, y: -120, w: 340, d: 240, r: 12 };
export const FEET = 10;
export const SLAB = 14;
export const TOP = FEET + SLAB;
export const WORKER_SCALE = 0.72;

/** The camera every Diagram is drawn from, as in the prototype. */
export const AZIMUTH = 45;

/** The glass conduit of a link: it lies on the floor and runs under each tile's slab to a riser. */
export const CONDUIT = 2.6;
export const RISER = 10;
const RISER_AT = { controller: 64, worker: 44 };

const RING_1 = 600;
const RING_2 = 1160;

// A ray at one of these angles clears the tiles' feet at both ends, and
// ring 2's rays pass between ring 1's tiles.
const SLOTS: readonly { readonly radius: number; readonly degrees: number }[] = [
  ...[-22.5, 157.5, 22.5, -157.5, -67.5, 112.5, 67.5, -112.5].map((degrees) => ({
    radius: RING_1,
    degrees,
  })),
  ...[0, 180, 45, -135, -45, 135].map((degrees) => ({ radius: RING_2, degrees })),
];

/** The most Nodes the Diagram draws: the Controller and a slot for each Worker. */
export const MOST_NODES = SLOTS.length + 1;

export type Placement = {
  readonly id: string;
  readonly at: Plane;
  readonly scale: number;
};

export type Route = {
  readonly worker: string;
  readonly points: readonly [Point3, Point3];
  /** The riser each end stands on, by its centre on the floor. */
  readonly risers: readonly [Plane, Plane];
  readonly degrees: number;
};

export type Layout = {
  /** Every tile, back to front. */
  readonly tiles: readonly Placement[];
  readonly links: readonly Route[];
  /** The Workers past MOST_NODES, which only the Roster lists. */
  readonly unplaced: number;
};

const depth = ([x, y]: Plane): number => x + y;

function routeTo(worker: string, at: Plane, degrees: number): Route {
  const radians = (degrees * Math.PI) / 180;
  const along = (distance: number): Plane => [
    Math.cos(radians) * distance,
    Math.sin(radians) * distance,
  ];
  const [x, y] = at;
  const fromController = RISER_AT.controller + RISER / 2;
  const fromWorker = (RISER_AT.worker + RISER / 2) * WORKER_SCALE;
  const length = Math.hypot(x, y);
  const [ax, ay] = along(fromController);
  const [bx, by] = along(length - fromWorker);
  const [cx, cy] = along(RISER_AT.controller);
  const [wx, wy] = along(length - RISER_AT.worker * WORKER_SCALE);
  return {
    worker,
    degrees,
    points: [
      [ax, ay, CONDUIT],
      [bx, by, CONDUIT],
    ],
    risers: [
      [cx, cy],
      [wx, wy],
    ],
  };
}

/**
 * Where each Node stands: the Controller at the centre, then each Worker in
 * pairing order on the slots around it, each linked to the Controller by a
 * straight conduit on the floor.
 */
export function layoutOf(controller: string, workers: readonly string[]): Layout {
  const placed = workers.slice(0, SLOTS.length);
  const tiles: Placement[] = [{ id: controller, at: [0, 0], scale: 1 }];
  const links = placed.map((worker, index) => {
    const slot = SLOTS[index];
    if (slot === undefined) {
      throw new Error(`no slot ${String(index)}`);
    }
    const radians = (slot.degrees * Math.PI) / 180;
    const at: Plane = [Math.cos(radians) * slot.radius, Math.sin(radians) * slot.radius];
    tiles.push({ id: worker, at, scale: WORKER_SCALE });
    return routeTo(worker, at, slot.degrees);
  });
  return {
    tiles: tiles.toSorted((a, b) => depth(a.at) - depth(b.at)),
    links,
    unplaced: workers.length - placed.length,
  };
}

/** The corners of a tile's plan, placed: for fitting and for depth. */
export function planOf({ at: [x, y], scale }: Placement): Plan {
  return {
    x: x + TILE.x * scale,
    y: y + TILE.y * scale,
    w: TILE.w * scale,
    d: TILE.d * scale,
    r: TILE.r * scale,
  };
}
