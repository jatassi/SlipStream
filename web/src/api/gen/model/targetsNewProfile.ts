export interface TargetsNewProfile {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  /**
   * @minLength 1
   * @maxLength 100
   */
  name: string;
  /**
   * Highest ranked first
   * @minItems 1
   */
  targetIds: string[];
}
