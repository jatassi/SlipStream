import type { Point3, Projection } from "./iso-kit";

export interface TubeOptions {
  readonly maxLength?: number;
  readonly spacing?: number;
  readonly closed?: boolean;
  readonly caps?: boolean | readonly [boolean, boolean];
  readonly stripes?: "auto" | boolean;
}

export interface TubePiece {
  readonly body: string;
  readonly shine: string;
  readonly shade: string;
  readonly edges: string;
  readonly wide: boolean;
  readonly mid: Point3;
  readonly points: readonly Point3[];
  readonly radii: readonly number[];
  readonly ends: readonly [boolean, boolean];
  readonly from: number;
  readonly to: number;
}

export function tubePieces(
  points: readonly Readonly<Point3>[],
  radius: number | ((at: number) => number),
  projection: Projection,
  options?: TubeOptions,
): TubePiece[];
