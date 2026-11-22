export interface BackupsSheetState {
  /**
   * When the recovery sheet was last downloaded; null before
   * @nullable
   */
  downloadedAt: string | null;
  /** The destinations it doesn't list, every one before a download */
  unlisted: string[];
}
