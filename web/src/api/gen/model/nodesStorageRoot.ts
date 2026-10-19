import type { NodesStorageRootState } from "./nodesStorageRootState.ts";

export interface NodesStorageRoot {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  id: string;
  /** The folder as its Node cleaned it */
  path: string;
  /** As listRoots names it */
  state: NodesStorageRootState;
  /** The volume its Node last found it ready on; absent before it did */
  volumeId?: string;
}
