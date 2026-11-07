export interface TransfersFile {
  name: string;
  /** Its job moved it to where it belongs */
  placed: boolean;
  /** @minimum 0 */
  position: number;
  /**
   * Bytes
   * @minimum 0
   */
  size: number;
  /** Every byte is verified at the destination */
  verified: boolean;
}
