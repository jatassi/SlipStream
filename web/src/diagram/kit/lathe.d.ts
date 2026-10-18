import type { Flat, Plane, Point3, Projection, SolidPaths } from "./iso-kit";

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

export interface Frame {
  readonly o: Point3;
  readonly a: Point3;
  readonly u: Point3;
  readonly v: Point3;
}

export interface Faded {
  readonly d: string;
  readonly alpha: number;
}

export type FadedDot = readonly [number, number] | readonly [number, number, number];

export function dot3(a: Readonly<Point3>, b: Readonly<Point3>): number;
export function mul3(a: Readonly<Point3>, k: number): Point3;
export function viewOf(projection: Projection): Point3;
export function frameAlong(o: Readonly<Point3>, a: Readonly<Point3>): Frame;
export function frameOf(
  o: Readonly<Point3>,
  a: Readonly<Point3>,
  u: Readonly<Point3>,
  v: Readonly<Point3>,
): Frame;
export function hullOf(points: readonly Readonly<Flat>[]): Flat[];
export function prismOf(
  frame: Frame,
  polygon: readonly Readonly<Plane>[],
  s0: number,
  s1: number,
  projection: Projection,
): SolidPaths;
export function disc(s0: number, s1: number, r: number, frame: Frame, projection: Projection): SolidPaths;
export function arcOf(frame: Frame, s: number, r: number, projection: Projection): string;
export function ribsOf(
  frame: Frame,
  s0: number,
  s1: number,
  r: number,
  count: number,
  projection: Projection,
  options: { readonly phase?: number; readonly fade: readonly [number, number] },
): Faded[];
export function dotsOf(
  frame: Frame,
  s: number,
  r: number,
  count: number,
  projection: Projection,
  options?: {
    readonly all?: boolean;
    readonly fade?: readonly [number, number];
  },
): FadedDot[];
