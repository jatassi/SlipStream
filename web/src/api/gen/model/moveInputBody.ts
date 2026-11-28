export interface MoveInputBody {
  /**
   * The Conversion it goes right after, in its lane; null for the head of Moved up
   * @nullable
   */
  after: string | null;
}
