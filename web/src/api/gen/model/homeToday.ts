export interface HomeToday {
  /**
   * Finished Imports, other than upgrades, that imported a file at or after the time in the since parameter
   * @minimum 0
   */
  imported: number;
  /**
   * Finished Imports that imported a file at or after the time in the since parameter and whose every file replaced one
   * @minimum 0
   */
  upgraded: number;
}
