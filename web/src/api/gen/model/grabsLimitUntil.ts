export interface GrabsLimitUntil {
  indexerId: string;
  indexerName: string;
  /** RSS sync uses all of its search limit, so Backlog search waits for it */
  starved: boolean;
  /** When its search limit frees a search */
  until: string;
}
