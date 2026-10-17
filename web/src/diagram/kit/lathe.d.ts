import type { Point3, Projection } from "./iso-kit";

export interface FilletOptions {
  readonly P?: Projection | null;
  readonly tube?: number;
  readonly margin?: number;
}

export function fillet(
  points: readonly Readonly<Point3>[],
  radius: number,
  perArc?: number,
  options?: FilletOptions,
): Point3[];
export function pathLength(points: readonly Readonly<Point3>[]): number;
export function pointAlong(points: readonly Readonly<Point3>[], at: number): Point3;
export function tangentAlong(points: readonly Readonly<Point3>[], at: number): Point3;
