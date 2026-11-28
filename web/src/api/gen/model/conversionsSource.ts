export interface ConversionsSource {
  /**
   * null once the source file is gone
   * @nullable
   */
  fileId: string | null;
  /** The source file's name */
  name: string;
  /** The name of the Target the source file fills */
  target: string;
}
