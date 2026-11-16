import type { GrabsChosenWhy } from "./grabsChosenWhy.ts";
import type { GrabsClientWait } from "./grabsClientWait.ts";
import type { GrabsEntered } from "./grabsEntered.ts";
import type { GrabsFailure } from "./grabsFailure.ts";
import type { GrabsFetchWait } from "./grabsFetchWait.ts";
import type { GrabsGrabActions } from "./grabsGrabActions.ts";
import type { GrabsGrabClient } from "./grabsGrabClient.ts";
import type { GrabsGrabFoundBy } from "./grabsGrabFoundBy.ts";
import type { GrabsGrabImport } from "./grabsGrabImport.ts";
import type { GrabsGrabQuality } from "./grabsGrabQuality.ts";
import type { GrabsGrabSource } from "./grabsGrabSource.ts";
import type { GrabsGrabStage } from "./grabsGrabStage.ts";
import type { GrabsGrabTarget } from "./grabsGrabTarget.ts";
import type { GrabsMetadataSearch } from "./grabsMetadataSearch.ts";
import type { GrabsProgress } from "./grabsProgress.ts";

export interface GrabsGrab {
  actions: GrabsGrabActions;
  /** Why the decision chose its Release; null when the ranking alone chose it */
  chosenBecause: GrabsChosenWhy | null;
  createdAt: string;
  /** Its Torrent's external Download Client, null for its Node's built-in client or until the .torrent is fetched */
  downloadClient: GrabsGrabClient | null;
  /** The Edition it was grabbed for: any when the Work takes any cut, else a cut key */
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
  /** What found its Release: a press of Search, Backlog search, RSS sync, a search again after a failed Grab, or an Admin in a Manual search */
  foundBy: GrabsGrabFoundBy;
  /**
   * A name, or the API key; null for a search's Grab
   * @nullable
   */
  grabbedBy: string | null;
  /** Each stage it entered, when it last entered it, oldest first */
  history: GrabsEntered[];
  id: string;
  /** Its download's Import, from importing on; null before */
  import: GrabsGrabImport | null;
  /** It is neither imported nor failed, so it counts in Activity and no other Grab can download the same files */
  inFlight: boolean;
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
  /** The ids of the Items its Release covers in the Work, in tree order */
  items: string[];
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
  /** The download's progress, absent unless it is downloading and its Node reported it */
  progress?: GrabsProgress;
  /** The quality the release title names, absent when it names none */
  quality?: GrabsGrabQuality;
  /** @minimum 0 */
  releaseSize: number;
  releaseTitle: string;
  /** Its Release in words, such as Bluray-1080p · FKW · 9.8 GB */
  releaseWords: string;
  /**
   * The Indexer whose feed RSS sync found it in; null unless foundBy is rss, and once the Indexer is deleted
   * @nullable
   */
  rssIndexerName: string | null;
  /**
   * The search run that made it; null for a Manual Grab and once the run is pruned
   * @nullable
   */
  searchId: string | null;
  /** How it was chosen: by a search's decision, by an Admin in a Manual search, or by RSS sync's decision */
  source: GrabsGrabSource;
  stage: GrabsGrabStage;
  /** The Targets it was grabbed for, in Profile order */
  targetIds: string[];
  /** Each Target of its Items it downloads for, in plan order */
  targets: GrabsGrabTarget[];
  /**
   * The Torrent it downloads, null until the .torrent is fetched
   * @nullable
   */
  torrentId: string | null;
  /**
   * The Tracker its release came from, null for a Grab older than Trackers
   * @nullable
   */
  trackerId: string | null;
  /** The client a fetching Grab goes to, absent unless it waits for that client: its Node to be back online, or a pinned client that can't take it */
  waitsFor?: GrabsClientWait;
  workId: string;
}
