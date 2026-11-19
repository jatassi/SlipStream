import { useEffect } from "react";

import type { NodesNode } from "../api/gen/model";
import { isWorkerRestarting, restartStore, useRestartState } from "../shell/restart/restart-state";
import type { Defines, Term } from "../ui/defined";
import type { Signal } from "../ui/signal-chip";
import type { Away } from "./standing";

/** A Node's state as the web draws it: its standing, with in-step read as online, or a restart the browser asked for. */
export type NodeState = "online" | Away | "restarting";

/** A Node's lamp is a signal colour, or the Work's glow while it restarts. */
export type Lamp = Signal | "live";

/** Every Node state's word, its lamp and, for a technical state, its definition (WRITING.md rule 25). */
export const nodeStates = {
  online: { word: "Online", lamp: "ok" },
  reconnecting: {
    word: "Reconnecting",
    lamp: "warn",
    definition:
      "Its connection to the Controller dropped a moment ago, and SlipStream retries automatically.",
  },
  offline: {
    word: "Offline",
    lamp: "warn",
    definition: "It isn't connected to the Controller, and its work waits until it's back online.",
  },
  lapsed: {
    word: "Offline too long",
    lamp: "fail",
    definition:
      "It has been offline for longer than the Offline grace, so the Controller does any of its work that can move.",
  },
  mismatched: {
    word: "On another version",
    lamp: "warn",
    definition:
      "It runs a different version of SlipStream from the Controller, and it gets no work until it's updated to the Controller's version.",
  },
  restarting: { word: "Restarting", lamp: "live" },
} as const satisfies Defines<NodeState, Term & { readonly lamp: Lamp }, "online" | "restarting">;

export const stateOf = (node: NodesNode): Exclude<NodeState, "restarting"> =>
  node.standing === "in-step" ? "online" : node.standing;

/**
 * The Workers restarting: each from the accepted request until its link is
 * up again. Once it is, the restart is cleared, so a later outage reads as
 * offline.
 */
export function useRestarting(nodes: readonly NodesNode[]): ReadonlySet<string> {
  const restarts = useRestartState();
  const restarting = new Set(
    nodes.filter((node) => isWorkerRestarting(restarts, node)).map(({ id }) => id),
  );
  const back = nodes
    .filter(({ id }) => restarts.workers.has(id) && !restarting.has(id))
    .map(({ id }) => id)
    .join(" ");
  useEffect(() => {
    for (const nodeId of back.split(" ").filter((id) => id !== "")) {
      restartStore.dispatch({ type: "back", nodeId });
    }
  }, [back]);
  return restarting;
}

export const useNodeState = (node: NodesNode): NodeState =>
  useRestarting([node]).has(node.id) ? "restarting" : stateOf(node);
