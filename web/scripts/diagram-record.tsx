import type { Point3, Projection } from "@anatomy/iso-kit";
import { renderToStaticMarkup } from "react-dom/server";

import * as A from "../vendor/anatomy/kit/audit.mjs";
import type { Item, Shape } from "../src/diagram/parts/part";
import { placedPipe, placedShape } from "../src/diagram/parts/placed";

function shapeOf(shape: Shape): A.AuditShape {
  const meta = { name: shape.name, ...(shape.owner === undefined ? {} : { owner: shape.owner }) };
  if (shape.kind === "slab") {
    return A.slab(shape.plan, shape.z, shape.h, meta, shape.steps);
  }
  if (shape.kind === "extruded") {
    return A.extruded(shape.ring, shape.z, shape.h, meta);
  }
  if (shape.kind === "prism") {
    return A.prism(shape.frame, shape.polygon, shape.s0, shape.s1, meta);
  }
  return A.disc(shape.frame, shape.s0, shape.s1, shape.r, meta);
}

/** Writes one line to standard output. */
export const say = (line: string): void => {
  process.stdout.write(`${line}\n`);
};

/**
 * Records an item drawn in local coordinates with its local projection, at
 * its place in the world: moved by `by`, scaled by `scale`.
 */
export function record(
  R: A.Recorder,
  item: Item,
  {
    by = [0, 0, 0],
    scale = 1,
    key,
  }: { readonly by?: Point3; readonly scale?: number; readonly key?: number } = {},
): void {
  const at: Point3 = [
    item.at[0] * scale + by[0],
    item.at[1] * scale + by[1],
    item.at[2] * scale + by[2],
  ];
  R.put({
    name: item.name,
    at,
    ...(key === undefined ? {} : { key }),
    shapes: [
      ...item.shapes.map((shape) => R.solid(shapeOf(placedShape(shape, by, scale)))),
      ...item.pipes.map((pipe) => {
        const moved = placedPipe(pipe, by, scale);
        return A.tube(moved.points, moved.r, { name: moved.name });
      }),
    ],
    svg: renderToStaticMarkup(<svg>{item.el}</svg>),
  });
  for (const pipe of item.pipes) {
    const moved = placedPipe(pipe, by, scale);
    R.route(moved.name, moved.points, moved.r, { owner: item.name });
  }
}

export const recorderFor = (P: Projection): A.Recorder => A.recorder(P);
