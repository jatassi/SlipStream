export interface RootsBusy {
  /**
   * The Imports into it that haven't finished
   * @minimum 0
   */
  imports: number;
  /**
   * The Moves from it or into it that haven't finished
   * @minimum 0
   */
  moves: number;
  /**
   * The Transfers into it or sending a file from it
   * @minimum 0
   */
  transfers: number;
}
