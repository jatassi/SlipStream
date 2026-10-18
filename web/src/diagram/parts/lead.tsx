import { type Plane, type Point3, type Projection, ring } from "@anatomy/iso-kit";
import { fillet } from "@anatomy/lathe";

import { type Drawn, type Item, type Pipe, item, line, post } from "./part";
import { tube } from "./tube";

/**
 * Where a machine's cable leaves it: a point on one of its faces, the floor
 * axis it runs out along, and how far it runs before it drops into the tile.
 */
export type Lead = {
  readonly from: Point3;
  readonly toward: Plane;
  readonly reach: number;
};

const CABLE = 0.8;
const GROMMET = { r: 2.4, h: 1.6, hole: 1.4 } as const;
const BEND = 2.4;

type Wiring = { readonly parts: readonly Drawn[]; readonly pipe: Pipe; readonly at: Plane };

/**
 * A machine's cable, in its local coordinates: out of its face, along the
 * floor axis, then down into a grommet set in the tile's top.
 */
export function wiring(P: Projection, name: string, { from, toward, reach }: Lead): Wiring {
  const [x, y, z] = from;
  const at: Plane = [x + toward[0] * reach, y + toward[1] * reach];
  const route: Point3[] = [from, [at[0], at[1], z], [at[0], at[1], GROMMET.h]];
  const cable = tube(P, `${name}.cable`, fillet(route, BEND, 6), CABLE, { tone: "lo" });
  return {
    at,
    pipe: cable.pipe,
    parts: [
      post(P, `${name}.grommet`, at, GROMMET.r, 0, GROMMET.h, { tone: "lo", steps: 20 }),
      line(ring(at[0], at[1], GROMMET.hole, GROMMET.h, P, 16), "faint"),
      cable,
    ],
  };
}

/** A machine's cable as an item of its own, sorted against the machine's parts. */
export function lead(P: Projection, name: string, spec: Lead): Item {
  const { at, parts, pipe } = wiring(P, name, spec);
  return item(name, [at[0], at[1], 0], parts, [pipe]);
}
