export interface MovesCanceledMoves {
  /**
   * Units canceled before their switch
   * @minimum 0
   */
  canceled: number;
  /**
   * Units that had already switched, which stay
   * @minimum 0
   */
  switched: number;
  /** The toast, such as Canceled 31 Moves. 12 movies already switched stay on den. */
  words: string;
}
