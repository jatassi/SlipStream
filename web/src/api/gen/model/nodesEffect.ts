import type { NodesDrains } from "./nodesDrains.ts";
import type { NodesGains } from "./nodesGains.ts";
import type { NodesHandedBack } from "./nodesHandedBack.ts";
import type { NodesLetsGo } from "./nodesLetsGo.ts";
import type { NodesMoves } from "./nodesMoves.ts";
import type { NodesPinned } from "./nodesPinned.ts";
import type { NodesWritesOff } from "./nodesWritesOff.ts";

export type NodesEffect =
  | NodesGains
  | NodesLetsGo
  | NodesDrains
  | NodesHandedBack
  | NodesPinned
  | NodesMoves
  | NodesWritesOff;
