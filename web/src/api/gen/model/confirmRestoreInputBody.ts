export interface ConfirmRestoreInputBody {
  /** Required for a Backup of another machine */
  oldControllerOff: boolean;
  /**
   * The Nodes to leave out of the restored library
   * @maxItems 10
   * @nullable
   */
  remove?: string[] | null;
}
