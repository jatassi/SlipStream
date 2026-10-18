declare module "*/vendor/anatomy/kit/audit.mjs" {
  import type { Plan, Plane, Point3, Projection } from "@anatomy/iso-kit";
  import type { Frame } from "@anatomy/lathe";

  export type AuditShape = { readonly name?: string };
  export type AuditItem = { readonly name: string; key: number };
  export type Settled = {
    readonly items: readonly AuditItem[];
    readonly overlaps: unknown;
    readonly solved: unknown;
    readonly states: readonly number[];
  };
  export type Recorder = {
    readonly P: Projection;
    readonly items: AuditItem[];
    settled: Settled | null;
    put(item: {
      readonly name: string;
      readonly at?: Point3;
      readonly key?: number;
      readonly shapes: readonly AuditShape[];
      readonly svg?: string;
    }): AuditItem;
    solid(shape: AuditShape): AuditShape;
    route(name: string, points: readonly Point3[], radius: number, options?: { readonly owner?: string }): unknown;
  };

  export function recorder(P: Projection): Recorder;
  export function slab(plan: Plan, z: number, height: number, meta: { readonly name: string; readonly owner?: string }, steps?: number): AuditShape;
  export function extruded(ring: readonly Plane[], z: number, height: number, meta: { readonly name: string; readonly owner?: string }): AuditShape;
  export function prism(
    frame: Frame,
    polygon: readonly Plane[],
    s0: number,
    s1: number,
    meta: { readonly name: string; readonly owner?: string },
  ): AuditShape;
  export function disc(frame: Frame, s0: number, s1: number, r: number, meta: { readonly name: string; readonly owner?: string }): AuditShape;
  export function tube(points: readonly Point3[], radius: number, meta: { readonly name: string; readonly owner?: string }): AuditShape;
  export function settle(R: Recorder): Settled;
  export function overlapsOf(
    items: readonly AuditItem[],
    P: Projection,
    options: { readonly states: readonly number[]; readonly skip: (a: AuditItem, b: AuditItem) => boolean },
  ): unknown;
  export function sameRun(a: AuditItem, b: AuditItem): boolean;
  export function auditOrExit(R: Recorder, P: Projection, options: { readonly argv: readonly string[] }): unknown;
}
