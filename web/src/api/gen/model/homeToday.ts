export interface HomeToday {
  /**
   * Finished Imports that landed a file since since, other than upgrades
   * @minimum 0
   */
  imported: number;
  /**
   * Finished Imports that landed a file since since and whose every file replaced one
   * @minimum 0
   */
  upgraded: number;
}
