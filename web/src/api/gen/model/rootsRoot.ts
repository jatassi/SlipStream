import type { RootsProgress } from "./rootsProgress.ts";
import type { RootsRootState } from "./rootsRootState.ts";

export interface RootsRoot {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  /** Where its Content Type's new Works go */
  default: boolean;
  /** The bytes of its files */
  fileBytes: number;
  id: string;
  /** The Storage Node whose disk holds it */
  nodeId: string;
  /** The folder as its Node cleaned it */
  path: string;
  /** The re-home checking its files at the new place now; absent while none is */
  rehome?: RootsProgress;
  /** Only ready takes imports. checking: its Node hasn't reported it yet; offline and written_off come from its Node's standing */
  state: RootsRootState;
  /** The volume its Node last found it ready on; absent before it did */
  volumeId?: string;
  /** The Works that live in it */
  works: number;
}
