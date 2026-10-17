import type { NodesOpenCode } from "./nodesOpenCode.ts";

export interface NodesOpenCodes {
  /** Every Join Code that may still pair a Node, oldest first */
  codes: NodesOpenCode[];
}
