export interface GrabsManualCell {
  /**
   * The Edition at the Item's Target: any, or a cut key
   * @minLength 1
   */
  edition: string;
  itemId: string;
  targetId: string;
}
