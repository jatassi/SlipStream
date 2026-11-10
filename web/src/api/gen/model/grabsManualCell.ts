export interface GrabsManualCell {
  /**
   * The cell's Edition: any, or a cut key
   * @minLength 1
   */
  edition: string;
  itemId: string;
  targetId: string;
}
