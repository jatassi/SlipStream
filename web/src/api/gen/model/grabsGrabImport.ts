export interface GrabsGrabImport {
  id: string;
  /**
   * Its files placed into the library
   * @minimum 0
   */
  placed: number;
  /**
   * Its files that are or were sending, placing, placed or waiting
   * @minimum 0
   */
  planned: number;
  /**
   * Its files a Transfer is sending
   * @minimum 0
   */
  sending: number;
  /**
   * Its files that wait for the Admin
   * @minimum 0
   */
  waiting: number;
  /** Its progress in words, such as Importing 6 of 10 episodes */
  words: string;
}
