export interface GrabsLimitUntil {
  indexerId: string;
  indexerName: string;
  /** RSS sync uses all of its search limit, so Backlog search and Search all wait for it */
  starved: boolean;
  /**
   * When its search limit frees a search; null while starved
   * @nullable
   */
  until: string | null;
}
