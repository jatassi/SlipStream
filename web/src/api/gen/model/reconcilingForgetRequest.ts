import type { ReconcilingForgetRequestFound } from "./reconcilingForgetRequestFound.ts";
import type { ReconcilingForgetRequestReason } from "./reconcilingForgetRequestReason.ts";

export interface ReconcilingForgetRequest {
  /** With no ids, how the group's files were found; left out, any */
  found?: ReconcilingForgetRequestFound;
  /**
   * The torrents; left out, every torrent in the group reason and found name
   * @maxItems 1000
   * @nullable
   */
  ids?: string[] | null;
  /** With no ids, the group's reason; left out, any */
  reason?: ReconcilingForgetRequestReason;
}
