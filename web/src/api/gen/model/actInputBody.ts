import type { ActInputBodyAction } from "./actInputBodyAction.ts";

export interface ActInputBody {
  action: ActInputBodyAction;
  /** remove: also delete each download copy */
  deleteFiles?: boolean;
  /** remove, mark-failed and recheck: return what it would do and change nothing */
  dryRun?: boolean;
  /** @minItems 1 */
  ids: string[];
}
