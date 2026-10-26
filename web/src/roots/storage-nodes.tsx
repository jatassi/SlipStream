import { Link } from "@tanstack/react-router";

import type { NodesNode } from "../api/gen/model";
import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import { NodeDot } from "../routes/_app/nodes/-state-dot";
import { Select } from "../ui/select";

const takesRoots = (node: NodesNode): boolean =>
  node.roles.some(
    ({ role, state }) =>
      role === "storage" && (state === "holding" || state === "away" || state === "covering"),
  );

/** The Nodes a root can go on, and the rest, which need Storage first. */
export function useStorageNodes() {
  const { nodes } = useListNodesSuspense().data;
  return {
    storage: nodes.filter((node) => takesRoots(node)),
    others: nodes.filter((node) => !takesRoots(node)),
  };
}

/**
 * The Storage Node a root goes on, each with its status dot, and how to give
 * the others Storage. With one Storage Node there is no choice to show, and
 * the root goes on it.
 */
export function NodeSelect({
  nodeId,
  onNodeChange,
}: {
  nodeId: string;
  onNodeChange: (nodeId: string) => void;
}) {
  const { storage, others } = useStorageNodes();
  return (
    <div className="flex flex-col gap-1.5">
      {storage.length > 1 && (
        <Select
          label="Node"
          options={storage.map((node) => ({
            value: node.id,
            label: node.name,
            leading: <NodeDot node={node} />,
          }))}
          value={nodeId}
          onValueChange={onNodeChange}
        />
      )}
      {others.map((node) => (
        <p key={node.id} className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
          To put a root on {node.name}, give it Storage in{" "}
          <Link
            to="/nodes/$nodeId/roles"
            params={{ nodeId: node.id }}
            className="text-ink-2 underline hover:text-ink-1"
          >
            its Roles
          </Link>
          .
        </p>
      ))}
    </div>
  );
}
