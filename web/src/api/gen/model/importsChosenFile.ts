export interface ImportsChosenFile {
  /**
   * The Items of the Work the file holds, in any order
   * @minItems 1
   */
  itemIds: string[];
  /**
   * A waiting file of the Import
   * @minimum 0
   */
  position: number;
}
