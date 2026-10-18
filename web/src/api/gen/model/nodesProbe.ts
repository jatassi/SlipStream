import type { NodesProbeFit } from "./nodesProbeFit.ts";
import type { NodesProbeProbe } from "./nodesProbeProbe.ts";
import type { NodesProbeReason } from "./nodesProbeReason.ts";

export interface NodesProbe {
  at: string;
  fit: NodesProbeFit;
  probe: NodesProbeProbe;
  /** Why it is weak or fails; absent when it fits */
  reason?: NodesProbeReason;
  /** What it found, as a sentence; absent when it fits */
  words?: string;
}
