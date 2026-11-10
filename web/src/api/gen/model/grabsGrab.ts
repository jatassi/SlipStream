import type { GrabsClientWait } from "./grabsClientWait.ts";
import type { GrabsEntered } from "./grabsEntered.ts";
import type { GrabsFailure } from "./grabsFailure.ts";
import type { GrabsFetchWait } from "./grabsFetchWait.ts";
import type { GrabsGrabActions } from "./grabsGrabActions.ts";
import type { GrabsGrabClient } from "./grabsGrabClient.ts";
import type { GrabsGrabQuality } from "./grabsGrabQuality.ts";
import type { GrabsGrabSource } from "./grabsGrabSource.ts";
import type { GrabsGrabStage } from "./grabsGrabStage.ts";
import type { GrabsMetadataSearch } from "./grabsMetadataSearch.ts";
import type { GrabsProgress } from "./grabsProgress.ts";

export interface GrabsGrab {
  actions: GrabsGrabActions;
  createdAt: string;
  /** Its Torrent's external Download Client, null for its Node's built-in client or until the .torrent is fetched */
  downloadClient: GrabsGrabClient | null;
  /** The Edition it was grabbed to fill: any under any cut, else a cut key */
  edition: string;
  /** Why it failed, absent unless it did */
  failure?: GrabsFailure;
  /**
   * The Node that fetched the .torrent: the Indexer's Searching Node, or the Downloading Node that got a magnet link's metadata from peers. Null until one did
   * @nullable
   */
  fetchNodeId: string | null;
  /** Why fetching the .torrent waits on its Indexer, absent unless it does */
  fetchWait?: GrabsFetchWait;
  /**
   * A name, or the API key; null for a search's Grab
   * @nullable
   */
  grabbedBy: string | null;
  /** Each stage it entered, when it last entered it, oldest first */
  history: GrabsEntered[];
  id: string;
  /**
   * The Indexer it was grabbed from, null once the Indexer is deleted
   * @nullable
   */
  indexerId: string | null;
  /**
   * The v1 info hash in hex, null until the .torrent is fetched
   * @nullable
   */
  infoHash: string | null;
  itemId: string;
  /**
   * The Manual search it came from; null for a search's Grab and once a later Manual search of the Work replaced it
   * @nullable
   */
  manualSearchId: string | null;
  /** The search for the magnet link's metadata, absent unless it is fetching from a magnet link and has asked its Downloading Node */
  metadata?: GrabsMetadataSearch;
  /** The Node that downloads it: the Indexer's home Node until the fetch, then the Downloading Node the fetch handed the .torrent to */
  nodeId: string;
  /** Each choice the Admin made past the decision, as a sentence */
  overrides: string[];
  /** The download's or the Transfer's progress, absent unless it is downloading and its Node reported it, or transferring */
  progress?: GrabsProgress;
  /** The quality the release title names, absent when it names none */
  quality?: GrabsGrabQuality;
  /** @minimum 0 */
  releaseSize: number;
  releaseTitle: string;
  /** How it was chosen: by a search's decision, or by an Admin in a Manual search */
  source: GrabsGrabSource;
  stage: GrabsGrabStage;
  /** The Targets it was grabbed for, in Profile order */
  targetIds: string[];
  /**
   * The Torrent it downloads, null until the .torrent is fetched
   * @nullable
   */
  torrentId: string | null;
  /**
   * The Tracker its Release reached, null for a Grab older than Trackers
   * @nullable
   */
  trackerId: string | null;
  /** The client a fetching Grab goes to, absent unless it waits for that client: its Node to be back online, or a pinned client that can't take it */
  waitsFor?: GrabsClientWait;
  workId: string;
}
