import type { RootsPlaceKind } from "./rootsPlaceKind.ts";

export interface RootsPlace {
  free: number;
  /** Absent for a root whose volume its Node hasn't read yet */
  kind?: RootsPlaceKind;
  name: string;
  path: string;
  /** The root this place is */
  rootId?: string;
  total: number;
}
