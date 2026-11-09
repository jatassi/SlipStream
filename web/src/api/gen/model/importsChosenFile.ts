export interface ImportsChosenFile {
  /**
   * The Item of the Work the file holds. One for now; S33 (Import pipeline) allows several
   * @minItems 1
   * @maxItems 1
   * @nullable
   */
  itemIds: string[] | null;
  /**
   * One of the pending import's paths
   * @minLength 1
   */
  path: string;
}
