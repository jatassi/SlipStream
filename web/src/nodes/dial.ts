import type { NodesNode } from "../api/gen/model";

/** Which side dialed the Node's live session: "seedbox dials home" or "home dials seedbox". */
export function reachedBy(controller: string, node: Pick<NodesNode, "name" | "dial">): string {
  if (node.dial.by === undefined) {
    return "Not connected";
  }
  return node.dial.by === "node"
    ? `${node.name} dials ${controller}`
    : `${controller} dials ${node.name}`;
}
