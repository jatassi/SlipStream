export interface RootsInUse {
  /** The files it holds, those waiting to be deleted counted */
  files: number;
  /** The Grabs in flight for its Works or into it */
  grabs: number;
}
