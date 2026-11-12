export interface FilesSpace {
  /**
   * Bytes free on the root's volume above its Storage error level
   * @minimum 0
   */
  free: number;
  /**
   * Bytes the file needs
   * @minimum 1
   */
  needs: number;
}
