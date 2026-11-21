import type { ListReconcilingTorrentsFound } from "./listReconcilingTorrentsFound.ts";
import type { ListReconcilingTorrentsReason } from "./listReconcilingTorrentsReason.ts";

export type ListReconcilingTorrentsParams = {
  /**
   * Only this group's reason; left out, any
   */
  reason?: ListReconcilingTorrentsReason;
  /**
   * Only torrents whose files were found so; left out, any
   */
  found?: ListReconcilingTorrentsFound;
  /**
   * The next of the previous page; left out, the first page
   * @maxLength 200
   */
  after?: string;
  /**
   * @minimum 1
   * @maximum 500
   */
  limit?: number;
};
