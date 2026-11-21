export interface ReconcilingSavedRoot {
  /**
   * How many recorded files aren't at the place, when that is why. Send it as forget to save anyway
   * @nullable
   */
  missing: number | null;
  /**
   * Why the root edit refused it, null once it is saved and its scan starts
   * @nullable
   */
  refused: string | null;
  rootId: string;
}
