import type { GrabsGroupState } from "./grabsGroupState.ts";
import type { GrabsLead } from "./grabsLead.ts";
import type { GrabsStopped } from "./grabsStopped.ts";
import type { GrabsWaiting } from "./grabsWaiting.ts";

export interface GrabsWorkSearching {
  /** Each search group of the Work, in tree order */
  groups: GrabsGroupState[];
  /** The one line the Work's header shows */
  lead: GrabsLead;
  /** Each cell SlipStream stopped searching again for after failed Grabs */
  stops: GrabsStopped[];
  /** Each cell waiting for a better Release, by when it grabs */
  waits: GrabsWaiting[];
}
