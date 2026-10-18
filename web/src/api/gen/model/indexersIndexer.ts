import type { IndexersIndexerProtocol } from "./indexersIndexerProtocol.ts";
import type { IndexersSearchMode } from "./indexersSearchMode.ts";
import type { IndexersWaiting } from "./indexersWaiting.ts";

export interface IndexersIndexer {
  /** False when the key was dropped and an Admin must enter it again */
  apiKeySet: boolean;
  createdAt: string;
  /** The Node that reaches the Indexer */
  homeNodeId: string;
  id: string;
  name: string;
  protocol: IndexersIndexerProtocol;
  /** By mode */
  searchModes: IndexersSearchMode[];
  url: string;
  /** Why its calls wait for its home Node, null while they go through */
  waiting: IndexersWaiting;
}
