export interface ImportsExtensionImpact {
  /**
   * The library files that stop counting as video
   * @minimum 0
   */
  files: number;
  /**
   * The files of unfinished Imports that stop counting as video
   * @minimum 0
   */
  pending: number;
}
