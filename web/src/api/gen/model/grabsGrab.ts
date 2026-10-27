import type { GrabsEntered } from "./grabsEntered.ts";
import type { GrabsFailure } from "./grabsFailure.ts";
import type { GrabsGrabQuality } from "./grabsGrabQuality.ts";
import type { GrabsGrabStage } from "./grabsGrabStage.ts";
import type { GrabsMetadataSearch } from "./grabsMetadataSearch.ts";
import type { GrabsProgress } from "./grabsProgress.ts";

export interface GrabsGrab {
  createdAt: string;
  /** The Edition it was grabbed to fill: any under any cut, else a cut key */
  edition: string;
  /** Why it failed, absent unless it did */
  failure?: GrabsFailure;
  /**
   * The Node that fetched the .torrent: the Indexer's Searching Node, or the Downloading Node that got a magnet link's metadata from peers. Null until one did
   * @nullable
   */
  fetchNodeId: string | null;
  /** Each stage it entered, when it last entered it, oldest first */
  history: GrabsEntered[];
  id: string;
  indexerId: string;
  /**
   * The v1 info hash in hex, null until the .torrent is fetched
   * @nullable
   */
  infoHash: string | null;
  itemId: string;
  /** The search for the magnet link's metadata, absent unless it is fetching from a magnet link and has asked its Downloading Node */
  metadata?: GrabsMetadataSearch;
  /** The Node that downloads it: the Indexer's home Node until the fetch, then the Downloading Node the fetch handed the .torrent to */
  nodeId: string;
  /** The download's or the Transfer's progress, absent unless it is downloading and its Node reported it, or transferring */
  progress?: GrabsProgress;
  /** The quality the release title names, absent when it names none */
  quality?: GrabsGrabQuality;
  /** @minimum 0 */
  releaseSize: number;
  releaseTitle: string;
  stage: GrabsGrabStage;
  /** The Targets it was grabbed for, in Profile order */
  targetIds: string[];
  /**
   * The Torrent it downloads, null until the .torrent is fetched
   * @nullable
   */
  torrentId: string | null;
  workId: string;
}
