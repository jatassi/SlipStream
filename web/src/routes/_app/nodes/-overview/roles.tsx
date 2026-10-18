import { Link } from "@tanstack/react-router";

import type { NodesNode } from "../../../../api/gen/model";
import { useGetRoleBoardSuspense } from "../../../../api/gen/query/nodes/nodes";
import type { Clock } from "../../../../format";
import { Chip } from "../../../../ui/chip";
import { roleChips } from "../-landing/view";

/** The Node's Roles as the Roster's chips, and the way to change them. */
export function Roles({
  node,
  nodes,
  clock,
}: {
  node: NodesNode;
  nodes: readonly NodesNode[];
  clock: Clock;
}) {
  const board = useGetRoleBoardSuspense().data;
  const chips = roleChips(
    { nodes },
    board,
    node.roles,
    { standing: node.standing, since: node.link.since },
    clock,
  );
  return (
    <span className="flex flex-wrap items-center gap-1.5">
      {chips.length === 0 && <span className="text-ink-3">No Roles yet</span>}
      {chips.map(({ role, words }) => (
        <Chip key={role} look="outline">
          {words}
        </Chip>
      ))}
      <Link
        to="/nodes/$nodeId/roles"
        params={{ nodeId: node.id }}
        className="ml-1 rounded-item text-meta text-ink-2 underline-offset-4 hover:text-ink-1 hover:underline"
      >
        Change Roles
      </Link>
    </span>
  );
}
