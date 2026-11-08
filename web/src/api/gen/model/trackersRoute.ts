import type { TrackersChoice } from "./trackersChoice.ts";
import type { TrackersTarget } from "./trackersTarget.ts";

export interface TrackersRoute {
  /** The home Node's built-in client; null while the home Node doesn't hold Downloading */
  default: TrackersTarget | null;
  /** The home Node of the Indexers that find the Grabs */
  homeNodeId: string;
  /** True while the stored client can't take a Grab because it is off or deleted, or its Node doesn't hold Downloading */
  skipped: boolean;
  /** The client an Admin picked; null for none */
  stored: TrackersChoice | null;
  /** Where the Grabs download now: the stored client while it can take them, else the default; null while they download where the Work's library root is */
  to: TrackersTarget | null;
}
