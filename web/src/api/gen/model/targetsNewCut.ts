export interface TargetsNewCut {
  /**
   * Other phrases that name it in a title; its name always does
   * @maxItems 20
   */
  aliases?: string[];
  /**
   * Its key is the name's words in lower case joined by hyphens, fixed once made
   * @minLength 1
   * @maxLength 100
   */
  name: string;
}
