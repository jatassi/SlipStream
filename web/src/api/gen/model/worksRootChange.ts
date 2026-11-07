export interface WorksRootChange {
  /** The root each Work moves to; a Work with files stays where it is */
  rootId: string;
  /**
   * A repeated id counts once
   * @minItems 1
   * @maxItems 20000
   */
  workIds: string[];
}
