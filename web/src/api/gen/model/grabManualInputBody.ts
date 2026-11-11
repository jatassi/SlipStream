import type { GrabsGrabPick } from "./grabsGrabPick.ts";

export interface GrabManualInputBody {
  manualSearchId: string;
  /**
   * @minItems 1
   * @maxItems 2000
   */
  picks: GrabsGrabPick[];
}
