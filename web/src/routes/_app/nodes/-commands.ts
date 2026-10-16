import { useQuery } from "@tanstack/react-query";

import { getListNodesQueryKey, listNodes } from "../../../api/gen/query/nodes/nodes";
import type { Command } from "../../../shell/commands";
import { nodeSections } from "./-sections";

/** The palette's Go to command for each section of the Node the browser talks to. */
export function useNodeSectionCommands(): Command[] {
  const self = useQuery({
    queryKey: getListNodesQueryKey(),
    queryFn: ({ signal }) => listNodes({ signal }),
  }).data?.nodes.find((node) => node.self);
  if (self === undefined) {
    return [];
  }
  return nodeSections.map((section) => ({
    id: `go-node-${section.id}`,
    label: `Go to ${section.label}`,
    meta: self.name,
    icon: section.icon,
    audience: "admin",
    run: ({ navigate }) => {
      void navigate({ to: section.to, params: { nodeId: self.id } });
    },
  }));
}
