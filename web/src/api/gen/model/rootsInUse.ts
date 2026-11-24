export interface RootsInUse {
  /** The files it holds, those waiting to be deleted counted */
  files: number;
  /** The Grabs in flight for its Works or into it */
  grabs: number;
  /** The Moves from it or into it that haven't finished */
  moves: number;
}
