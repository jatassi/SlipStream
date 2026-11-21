export interface SubtitlesUpload {
  /** The file, in base64 */
  bytes: string;
  /**
   * The file's name, whose extension says its format
   * @minLength 1
   * @maxLength 255
   */
  name: string;
}
