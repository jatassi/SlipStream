export interface GrabsSearchChanged {
  /**
   * The Manual search the run belongs to; null for an automatic run
   * @nullable
   */
  manualSearchId: string | null;
  searchId: string;
  workId: string;
}
