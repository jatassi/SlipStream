import type { NodesDialViewBy, NodesNode, NodesTopology } from "../api/gen/model";

/** Which side dialed the Node's live session: "seedbox dials home" or "home dials seedbox". */
export function reachedBy(
  controller: string,
  node: { readonly name: string; readonly dial: { readonly by?: NodesDialViewBy | undefined } },
): string {
  if (node.dial.by === undefined) {
    return "Not connected";
  }
  return node.dial.by === "node"
    ? `${node.name} dials ${controller}`
    : `${controller} dials ${node.name}`;
}

/** The side that dialed a Worker's live session in Topology, as listNodes reports it. */
export function dialOf(topology: NodesTopology, nodeId: string): NodesNode["dial"]["by"] {
  const dialer = topology.links.find(({ b }) => b === nodeId)?.dialer;
  if (dialer === undefined) {
    return undefined;
  }
  return dialer === nodeId ? "node" : "controller";
}
