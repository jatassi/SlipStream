export type Flat = [number, number];
export type Point3 = [number, number, number];
export type Plane = [number, number];

export interface Projection {
  readonly origin: Flat;
  readonly scale: number;
  readonly azimuth?: number;
  readonly elevation?: number;
}

export interface Plan {
  readonly x: number;
  readonly y: number;
  readonly w: number;
  readonly d: number;
  readonly r: number;
}

export interface SolidPaths {
  readonly fill: string;
  readonly outline: string;
  readonly crease: string;
  readonly top: string;
  readonly shades: readonly string[];
  readonly bevel?: string;
  readonly seamless?: boolean;
}

export type Tone = "hi" | "mid" | "lo" | "faint";
export type LineTone = Tone | "lit" | "red" | "green" | "blue";
export type KeyMark =
  | "raised"
  | "flat"
  | "lit"
  | "edge"
  | "dim"
  | "dotted"
  | "dashed"
  | "red"
  | "green"
  | "blue";

export interface LegendKey {
  readonly mark: KeyMark;
  readonly label: string;
}

export interface FitOptions {
  readonly pad?: number;
  readonly headroom?: number;
  readonly azimuth?: number;
  readonly elevation?: number;
}

export const AZIMUTH: number;
export const ELEVATION: number;

export function iso(point: Readonly<Point3>, projection: Projection): Flat;
export function depthOf(point: Readonly<Point3>, projection: Projection): number;
export function pathOf(points: readonly Readonly<Flat>[], closed?: boolean): string;
export function fitProjection(
  points: readonly Readonly<Point3>[],
  width: number,
  height: number,
  options?: FitOptions,
): Projection;
export function slabOf(
  plan: Plan,
  z: number,
  height: number,
  projection: Projection,
  steps?: number,
  bevel?: number,
): SolidPaths;
export function insetPlan(plan: Plan, by: number): Plan;
export function onTop(
  points: readonly Readonly<Plane>[],
  z: number,
  projection: Projection,
  closed?: boolean,
): string;
export function planOutline(plan: Plan, z: number, projection: Projection, steps?: number): string;
export function topMatrix(
  at: Readonly<Plane>,
  z: number,
  projection: Projection,
  along?: "x" | "y",
): string;

export interface TickLines {
  readonly minor: string;
  readonly major: string;
}

export function towardViewer(projection: Projection): Plane;
export function roundedPlan(plan: Plan, steps?: number): Plane[];
export function ringOf(plan: Plan, steps?: number): Plane[];
export function extrude(
  ring: readonly Readonly<Plane>[],
  z: number,
  height: number,
  projection: Projection,
  options?: { readonly convex?: boolean; readonly bevel?: number },
): SolidPaths;
export function cylinder(
  cx: number,
  cy: number,
  radius: number,
  z: number,
  height: number,
  projection: Projection,
  steps?: number,
  bevel?: number,
): SolidPaths;
export function corners(plan: Plan, inset?: number): Plane[];
export function boxCorners(plan: Plan, z0: number, z1: number): Point3[];
export function haloOf(plan: Plan, z: number, height: number, projection: Projection): string;
export function lineOnTop(
  from: Readonly<Plane>,
  to: Readonly<Plane>,
  z: number,
  projection: Projection,
): string;
export function segment(
  from: Readonly<Point3>,
  to: Readonly<Point3>,
  projection: Projection,
): string;
export function ring(
  cx: number,
  cy: number,
  radius: number,
  z: number,
  projection: Projection,
  steps?: number,
): string;
export function dotGrid(
  x: number,
  y: number,
  columns: number,
  rows: number,
  gap: number,
  z: number,
  projection: Projection,
): Flat[];
export function knurl(
  cx: number,
  cy: number,
  radius: number,
  z0: number,
  z1: number,
  count: number,
  projection: Projection,
  inset?: number,
): string;
export function sideArc(
  cx: number,
  cy: number,
  radius: number,
  z: number,
  projection: Projection,
  steps?: number,
): string;
export function sideSeam(plan: Plan, z: number, projection: Projection, steps?: number): string;
export function topTicks(
  from: number,
  to: number,
  step: number,
  every: number,
  edge: number,
  z: number,
  lengths: readonly [number, number],
  projection: Projection,
  axis?: "x" | "y",
): TickLines;
export function translateAlong(projection: Projection, by: Readonly<Point3>): string;
export function sideMatrix(
  at: Readonly<Plane>,
  z: number,
  projection: Projection,
  face: "left" | "right",
): string;
