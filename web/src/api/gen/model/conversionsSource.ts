export interface ConversionsSource {
  /** The cut of the cell it fills */
  edition: string;
  /**
   * null once the source file is gone
   * @nullable
   */
  fileId: string | null;
  /** The source file's name */
  name: string;
  /** The name of the Target the source file fills; empty once no Profile makes this Target from one */
  target: string;
  /**
   * The Target the source file fills; null once no Profile makes this Target from one
   * @nullable
   */
  targetId: string | null;
}
