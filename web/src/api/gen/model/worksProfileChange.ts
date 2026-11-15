export interface WorksProfileChange {
  /** The Profile to give each Work; a Work with its own Edition list keeps it */
  profileId: string;
  /**
   * A repeated id counts once
   * @minItems 1
   * @maxItems 20000
   */
  workIds: string[];
}
