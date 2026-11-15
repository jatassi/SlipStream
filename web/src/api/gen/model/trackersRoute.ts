import type { TrackersChoice } from "./trackersChoice.ts";
import type { TrackersTarget } from "./trackersTarget.ts";

export interface TrackersRoute {
  /** The home Node's built-in client; null while the home Node doesn't have the Downloading Role */
  default: TrackersTarget | null;
  /** The home Node of the Indexers that find the Grabs */
  homeNodeId: string;
  /** True while the stored client can't take a Grab because it is off or deleted, or its Node doesn't have the Downloading Role */
  skipped: boolean;
  /** The client an Admin picked; null for none */
  stored: TrackersChoice | null;
  /** Where the Grabs download now: the stored client while it can take them, else the default; null while each goes to the client set for its Work's library root */
  to: TrackersTarget | null;
}
