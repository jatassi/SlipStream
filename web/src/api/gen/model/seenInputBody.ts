export interface SeenInputBody {
  /**
   * The seq of the newest row the Admin has seen
   * @minimum 0
   */
  through: number;
}
