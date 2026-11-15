export type DeleteDownloadClientParams = {
  /**
   * Delete a client that has torrents SlipStream tracks: they stay in the client, and Grabs not yet imported from them fail
   */
  stopTracking?: boolean;
};
