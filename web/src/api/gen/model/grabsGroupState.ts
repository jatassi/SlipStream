import type { GrabsBehind } from "./grabsBehind.ts";
import type { GrabsGroupStateLane } from "./grabsGroupStateLane.ts";
import type { GrabsGroupStateLastResult } from "./grabsGroupStateLastResult.ts";
import type { GrabsGroupStateState } from "./grabsGroupStateState.ts";
import type { GrabsGroupStateTrigger } from "./grabsGroupStateTrigger.ts";

export interface GrabsGroupState {
  /** Queued again behind its own running search */
  again: boolean;
  /** What it waits on; null when nothing holds it */
  behind: GrabsBehind | null;
  /** The search group's key, to compare and never to parse */
  key: string;
  /** The group in words, such as season 3 */
  label: string;
  /**
   * The queued entry's lane, else the running search's; null while idle
   * @nullable
   */
  lane: GrabsGroupStateLane;
  /**
   * The Releases its last search found
   * @minimum 0
   */
  lastFound: number;
  /**
   * The Grabs its last search made or joined
   * @minimum 0
   */
  lastGrabs: number;
  /**
   * How its last search finished; null before one did
   * @nullable
   */
  lastResult: GrabsGroupStateLastResult;
  /**
   * When its last search finished; null before one did
   * @nullable
   */
  lastSearchedAt: string | null;
  /**
   * When Backlog search searches it next; null while queued or searching, and while Backlog search is off
   * @nullable
   */
  nextSearchAt: string | null;
  state: GrabsGroupStateState;
  /**
   * What queued it, else what started the running search; null while idle
   * @nullable
   */
  trigger: GrabsGroupStateTrigger;
}
