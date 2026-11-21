import type { ReconcilingGroup } from "./reconcilingGroup.ts";
import type { ReconcilingNodeRow } from "./reconcilingNodeRow.ts";
import type { ReconcilingRestore } from "./reconcilingRestore.ts";
import type { ReconcilingRootRow } from "./reconcilingRootRow.ts";
import type { ReconcilingUnknown } from "./reconcilingUnknown.ts";

export interface ReconcilingPage {
  /** Each Node Reconciling waits on, or whose adoption gave it the Downloading Role */
  nodes: ReconcilingNodeRow[];
  /** The restore Reconciling runs for; null outside one, when the page lists only the Nodes that keep their torrents */
  restore: ReconcilingRestore | null;
  /** The review list's groups, with how many torrents each holds */
  review: ReconcilingGroup[];
  /** Each library root Reconciling waits on */
  roots: ReconcilingRootRow[];
  /** Nodes not in this Backup that tried to connect lately, newest last; empty outside a restore */
  unknown: ReconcilingUnknown[];
}
