import { type Flat, type Projection, pathOf, roundedPlan, sideMatrix } from "@anatomy/iso-kit";

/** A rounded rectangle on a face, from (u0, z0) to (u1, z1), in a face's own coordinates. */
export const roundRect = (
  u0: number,
  z0: number,
  u1: number,
  z1: number,
  r: number,
  steps = 4,
): string =>
  pathOf(
    roundedPlan({ x: u0, y: z0, w: u1 - u0, d: z1 - z0, r }, steps).map(([u, z]): Flat => [u, -z]),
    true,
  );

/** The transform onto the face at y, looking from +y: u runs along x, up is z. */
export const faceY = (P: Projection, y: number): string => sideMatrix([0, y], 0, P, "left");
/** The transform onto the face at x, looking from +x: u runs along -y, up is z. */
export const faceX = (P: Projection, x: number): string => sideMatrix([x, 0], 0, P, "right");
