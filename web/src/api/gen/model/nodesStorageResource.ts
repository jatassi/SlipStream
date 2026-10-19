import type { NodesStorageRoot } from "./nodesStorageRoot.ts";
import type { NodesStorageVolume } from "./nodesStorageVolume.ts";

export interface NodesStorageResource {
  /** The library roots on the Node, by Content Type, oldest first in each */
  roots: NodesStorageRoot[];
  /** The volumes its roots sit on */
  volumes: NodesStorageVolume[];
}
