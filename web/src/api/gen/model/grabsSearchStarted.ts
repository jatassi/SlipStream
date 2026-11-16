export interface GrabsSearchStarted {
  /** One run per Item in the scope with a missing, failed or upgradable file */
  searchIds: string[];
}
