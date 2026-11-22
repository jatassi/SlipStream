import type { ReconcilingReaddRequestFound } from "./reconcilingReaddRequestFound.ts";
import type { ReconcilingReaddRequestReason } from "./reconcilingReaddRequestReason.ts";

export interface ReconcilingReaddRequest {
  /** With no ids, how the group's files were found; left out, any */
  found?: ReconcilingReaddRequestFound;
  /**
   * The torrents; left out, every torrent in the group reason and found name
   * @maxItems 1000
   * @nullable
   */
  ids?: string[] | null;
  /** The Node to re-add them on; left out, each on the Node it was recorded on */
  nodeId?: string;
  /** With no ids, the group's reason; left out, any */
  reason?: ReconcilingReaddRequestReason;
}
