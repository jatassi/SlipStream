import { Link } from "@tanstack/react-router";

import type { NodesNode } from "../../api/gen/model";
import { useListIndexersSuspense } from "../../api/gen/query/indexers/indexers";
import { useListNodesSuspense } from "../../api/gen/query/nodes/nodes";
import { Empty } from "../../shell/empty";
import { Section } from "../../ui/section";
import { UngroupedIndexers } from "../tracker-group";

export function HomedIndexers({ node }: { node: NodesNode }) {
  const homed = useListIndexersSuspense().data.indexers.filter(
    (indexer) => indexer.homeNodeId === node.id,
  );
  const { nodes } = useListNodesSuspense().data;
  return (
    <Section title="Indexers" count={homed.length}>
      {homed.length === 0 ? (
        <Empty
          action={
            <Link
              to="/settings/$section"
              params={{ section: "indexers" }}
              className="text-ink-1 underline"
            >
              Indexers
            </Link>
          }
        >
          No Indexer has {node.name} as its home Node.
        </Empty>
      ) : (
        <UngroupedIndexers indexers={homed} nodes={nodes} />
      )}
    </Section>
  );
}
