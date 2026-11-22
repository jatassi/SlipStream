import type {
  NodesNode,
  ReconcilingNodeRow,
  ReconcilingPage,
  ReconcilingRestore,
} from "../api/gen/model";
import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import type { Clock } from "../ui/clock";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { List, ListRow } from "../ui/list";
import { Section } from "../ui/section";
import { ContinueWithout } from "./continue";
import { PairButton } from "./pair-dialog";
import { type NodeState, nodeLine, nodeState, nodeWaits } from "./words";

const continuable = new Set<NodeState>([
  "pair",
  "lapsed",
  "offline",
  "waiting",
  "refused",
  "matching",
]);

function NodeActions({
  row,
  state,
  restoring,
}: {
  row: ReconcilingNodeRow;
  state: NodeState;
  restoring: boolean;
}) {
  const pair = state === "pair";
  const without = restoring && continuable.has(state);
  if (!pair && !without) {
    return null;
  }
  return (
    <div className="flex flex-wrap gap-2 pt-1">
      {pair && <PairButton title={`Pair ${row.name} again`} node={row.name} />}
      {without && <ContinueWithout node={row} look={state === "lapsed" ? "primary" : "default"} />}
    </div>
  );
}

function NodeMachinery({ row, state }: { row: ReconcilingNodeRow; state: NodeState }) {
  if (state === "refused" && row.refused !== null) {
    return (
      <Details>
        <p>{row.refused}</p>
      </Details>
    );
  }
  if (state === "pair") {
    return (
      <Details>
        <p>
          Create a Join Code, then on {row.name} run <Data>slipstream join</Data> with it. Or open{" "}
          {row.name} in a browser, enter the Setup code from its log, then the Join Code.
        </p>
      </Details>
    );
  }
  return null;
}

function NodeRow({
  row,
  restore,
  clock,
}: {
  row: ReconcilingNodeRow;
  restore: ReconcilingRestore | null;
  clock: Clock;
}) {
  const state = nodeState(row, restore?.newMachine ?? false);
  return (
    <ListRow
      title={row.name}
      wrap
      detail={
        <div className="flex flex-col gap-2 pb-1 text-ink-2">
          <p>{nodeLine(row, state, clock)}</p>
          <NodeMachinery row={row} state={state} />
          <NodeActions row={row} state={state} restoring={restore !== null} />
        </div>
      }
    />
  );
}

const relays = (node: NodesNode | undefined): boolean =>
  node?.roles.some(({ role }) => role === "relaying") ?? false;

/** Nodes Reconciling waits on come first, and of those, Nodes that hold Relaying. */
function ordered(
  rows: readonly ReconcilingNodeRow[],
  nodes: readonly NodesNode[],
): ReconcilingNodeRow[] {
  const byId = new Map(nodes.map((node) => [node.id, node]));
  const rank = (row: ReconcilingNodeRow) =>
    (nodeWaits(row) ? 0 : 2) + (relays(byId.get(row.id)) ? 0 : 1);
  return rows.toSorted((a, b) => rank(a) - rank(b));
}

export function NodesSection({ page, clock }: { page: ReconcilingPage; clock: Clock }) {
  const { nodes } = useListNodesSuspense().data;
  if (page.nodes.length === 0) {
    return null;
  }
  return (
    <Section title="Nodes">
      <List label="Nodes">
        {ordered(page.nodes, nodes).map((row) => (
          <NodeRow key={row.id} row={row} restore={page.restore} clock={clock} />
        ))}
      </List>
    </Section>
  );
}
