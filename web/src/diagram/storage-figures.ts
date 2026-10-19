import type { NodesStorageResource } from "../api/gen/model";

/** How many volumes the Storage cabinet shows, one to a drawer. */
export const DRAWERS = 4;

const usedOf = ({ total, free }: NodesStorageResource["volumes"][number]): number =>
  total > 0 ? 1 - free / total : 0;

/** The used share of each volume a Node's roots sit on, fullest first, one per drawer. */
export const drawersOf = (storage: NodesStorageResource | undefined): readonly number[] =>
  (storage?.volumes ?? [])
    .map((volume) => usedOf(volume))
    .toSorted((a, b) => b - a)
    .slice(0, DRAWERS);
