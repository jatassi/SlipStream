export interface ReachabilityCounts {
  /** Fixable, resting on a failing check */
  fix: number;
  /** Reachable by what SlipStream sees here, not tested from outside */
  lookRight: number;
  /** Needs a relay */
  relay: number;
  /** Fixable, resting on Not tested */
  untested: number;
  /** Reachable and tested from outside */
  work: number;
}
