import { type Plane, type Point3, type Projection, iso } from "@anatomy/iso-kit";
import type { Frame } from "@anatomy/lathe";

import type { Item, Pipe, Shape } from "./part";

/** The projection that draws local coordinates at `by` in the world. */
export const movedP = (P: Projection, by: Point3): Projection => ({ ...P, origin: iso(by, P) });

const add = ([x, y, z]: Point3, [dx, dy, dz]: Point3): Point3 => [x + dx, y + dy, z + dz];

/** A shape moved by `by` and scaled about the origin by `scale`, as the audit places it in the world. */
export function placedShape(shape: Shape, by: Point3, scale = 1): Shape {
  const [dx, dy, dz] = by;
  const at = (point: Point3): Point3 =>
    add([point[0] * scale, point[1] * scale, point[2] * scale], by);
  const frame = (f: Frame): Frame => ({ ...f, o: at(f.o) });
  if (shape.kind === "slab") {
    const { plan } = shape;
    return {
      ...shape,
      plan: {
        x: plan.x * scale + dx,
        y: plan.y * scale + dy,
        w: plan.w * scale,
        d: plan.d * scale,
        r: plan.r * scale,
      },
      z: shape.z * scale + dz,
      h: shape.h * scale,
    };
  }
  if (shape.kind === "extruded") {
    return {
      ...shape,
      ring: shape.ring.map(([x, y]): Plane => [x * scale + dx, y * scale + dy]),
      z: shape.z * scale + dz,
      h: shape.h * scale,
    };
  }
  if (shape.kind === "prism") {
    return {
      ...shape,
      frame: frame(shape.frame),
      polygon: shape.polygon.map(([u, v]): Plane => [u * scale, v * scale]),
      s0: shape.s0 * scale,
      s1: shape.s1 * scale,
    };
  }
  return {
    ...shape,
    frame: frame(shape.frame),
    s0: shape.s0 * scale,
    s1: shape.s1 * scale,
    r: shape.r * scale,
  };
}

export const placedPipe = (pipe: Pipe, by: Point3, scale = 1): Pipe => ({
  ...pipe,
  points: pipe.points.map((point) =>
    add([point[0] * scale, point[1] * scale, point[2] * scale], by),
  ),
  r: pipe.r * scale,
});

/** An item drawn in local coordinates, with its shapes and anchor moved by `by` into its parent's. */
export const placed = (it: Item, by: Point3): Item => ({
  ...it,
  at: add(it.at, by),
  shapes: it.shapes.map((shape) => placedShape(shape, by)),
  pipes: it.pipes.map((pipe) => placedPipe(pipe, by)),
});
