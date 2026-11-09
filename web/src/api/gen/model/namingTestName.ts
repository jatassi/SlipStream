export interface NamingTestName {
  /**
   * A file name, or a path inside the Work's folder
   * @minLength 1
   */
  name: string;
  /** The Work the name matches, as POST /releases/parse with kind file reads it */
  workId: string;
}
