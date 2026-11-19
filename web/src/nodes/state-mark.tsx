import type { NodesNode } from "../api/gen/model";
import { Defined } from "../ui/defined";
import { LiveDot } from "../ui/live-chip";
import type { Signal } from "../ui/signal-chip";
import { type NodeState, nodeStates, useNodeState } from "./state";

const lamps: { readonly [S in Signal]: string } = {
  ok: "bg-ok",
  warn: "bg-warn",
  fail: "bg-fail",
};

/** A Node's state as its 6px lamp, which says its word to a screen reader unless `decorative`, because the word sits beside it. */
export function NodeLamp({
  state,
  decorative = false,
}: {
  state: NodeState;
  decorative?: boolean;
}) {
  const { word, lamp } = nodeStates[state];
  return (
    <span className="flex shrink-0 items-center">
      {lamp === "live" ? (
        <LiveDot />
      ) : (
        <span aria-hidden="true" className={`size-1.5 rounded-capsule ${lamps[lamp]}`} />
      )}
      {!decorative && <span className="sr-only">{word}</span>}
    </span>
  );
}

/** The lamp of a Node's live state, restart included, for inside a control such as an option. */
export function NodeDot({ node }: { node: NodesNode }) {
  return <NodeLamp state={useNodeState(node)} />;
}

/** A Node's lamp and its word. A technical state's word opens its definition on hover, tap and Enter, so it never sits inside a control. */
export function NodeStateMark({ state }: { state: NodeState }) {
  return (
    <Defined term={nodeStates[state]}>
      {(word) => (
        <span className="inline-flex items-center gap-2 text-ink-2">
          <NodeLamp state={state} decorative />
          {word}
        </span>
      )}
    </Defined>
  );
}
