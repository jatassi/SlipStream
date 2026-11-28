import type { ConversionsLiveNode } from "./conversionsLiveNode.ts";
import type { ConversionsLiveRow } from "./conversionsLiveRow.ts";

export interface ConversionsLiveSample {
  nodes: ConversionsLiveNode[];
  rows: ConversionsLiveRow[];
}
