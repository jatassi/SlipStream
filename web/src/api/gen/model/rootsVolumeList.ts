import type { RootsNodeVolumes } from "./rootsNodeVolumes.ts";

export interface RootsVolumeList {
  /** Each Storage Node with a volume that holds a root, in id order */
  nodes: RootsNodeVolumes[];
}
