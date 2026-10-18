import type { NodesNode } from "../../../api/gen/model";
import { type NodeState, stateWords, useNodeState } from "./-node";

const colors: { readonly [S in NodeState]: string } = {
  online: "bg-ok",
  reconnecting: "bg-ink-4",
  offline: "bg-warn",
  lapsed: "bg-fail",
  mismatched: "bg-warn",
  restarting: "bg-ink-1 shadow-live-glow",
};

/** A Node's state as a dot, with its word for a screen reader unless the words sit beside it. */
export function StateDot({ state, worded = false }: { state: NodeState; worded?: boolean }) {
  return (
    <span className="flex shrink-0 items-center">
      <span aria-hidden="true" className={`size-1.5 rounded-capsule ${colors[state]}`} />
      {!worded && <span className="sr-only">{stateWords[state]}</span>}
    </span>
  );
}

export function NodeDot({ node }: { node: NodesNode }) {
  return <StateDot state={useNodeState(node)} />;
}
