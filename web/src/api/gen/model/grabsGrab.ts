import type { GrabsEntered } from "./grabsEntered.ts";
import type { GrabsFailure } from "./grabsFailure.ts";
import type { GrabsGrabEdition } from "./grabsGrabEdition.ts";
import type { GrabsGrabStage } from "./grabsGrabStage.ts";
import type { GrabsProgress } from "./grabsProgress.ts";

export interface GrabsGrab {
  createdAt: string;
  edition: GrabsGrabEdition;
  /** Why it failed, null unless it did */
  failure: GrabsFailure;
  /** Each stage it entered, oldest first */
  history: GrabsEntered[];
  id: string;
  indexerId: string;
  /**
   * The v1 info hash in hex, null until the .torrent is fetched
   * @nullable
   */
  infoHash: string | null;
  itemId: string;
  /** The Node that downloads it: the Indexer's home Node */
  nodeId: string;
  /** The download's progress, null unless it is downloading and its Node reported it */
  progress: GrabsProgress;
  /** @minimum 0 */
  releaseSize: number;
  releaseTitle: string;
  stage: GrabsGrabStage;
  /** The Targets it was grabbed for, in Profile order */
  targetIds: string[];
  workId: string;
}
