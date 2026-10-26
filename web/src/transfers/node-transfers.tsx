import type { NodesNode } from "../api/gen/model";
import { useListTransfersSuspense } from "../api/gen/query/transfers/transfers";
import { Section } from "../ui/section";
import { StatusSentence } from "../ui/status-sentence";
import { SendLimit } from "./send-limit";
import { Speeds } from "./speeds";
import { TransferRow } from "./transfer-row";
import { statusOf } from "./words";

const listId = "transfers";

/** Nodes > a Node > Transfers: what the Node sends and receives, its send limit and its recent speeds. */
export function NodeTransfers({ node }: { node: NodesNode }) {
  const { transfers } = useListTransfersSuspense({ nodeId: node.id }).data;
  return (
    <>
      <div className="mb-6">
        <StatusSentence parts={statusOf(transfers, node.id, `#${listId}`)} />
      </div>
      <SendLimit nodeId={node.id} />
      {transfers.length > 0 && (
        <Section id={listId} title="Transfers" count={transfers.length}>
          <ul
            aria-label={`Transfers of ${node.name}`}
            className="rounded-card bg-glass ring-1 ring-hairline ring-inset"
          >
            {transfers.map((transfer) => (
              <TransferRow key={transfer.id} transfer={transfer} nodeId={node.id} />
            ))}
          </ul>
        </Section>
      )}
      <Speeds node={node} />
    </>
  );
}
