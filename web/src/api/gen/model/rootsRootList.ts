import type { RootsRoot } from "./rootsRoot.ts";

export interface RootsRootList {
  /** By Content Type, oldest first in each */
  roots: RootsRoot[];
}
