import type { IndexersIndexerProtocol } from "./indexersIndexerProtocol.ts";
import type { IndexersSearchMode } from "./indexersSearchMode.ts";

export interface IndexersIndexer {
  createdAt: string;
  /** The Node that reaches the Indexer */
  homeNodeId: string;
  id: string;
  name: string;
  protocol: IndexersIndexerProtocol;
  /** By mode */
  searchModes: IndexersSearchMode[];
  url: string;
}
