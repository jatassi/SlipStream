import type { ReactNode } from "react";

import { useListNodesSuspense } from "../../../api/gen/query/nodes/nodes";
import { useShell } from "../../../ui/shell-context";
import { nodeOf } from "./-node";
import { PhoneNodePicker } from "./-node-picker";
import { SectionList } from "./-section-list";

/** A Node's top page. On phone it carries the Node picker and lists the other sections. */
export function Overview({ nodeId, children }: { nodeId: string; children: ReactNode }) {
  const { nodes } = useListNodesSuspense().data;
  const node = nodeOf(nodes, nodeId);
  const phone = useShell() === "phone";
  return (
    <>
      {phone && nodes.length > 1 && <PhoneNodePicker nodes={nodes} current={node} />}
      {children}
      {phone && <SectionList node={node} />}
    </>
  );
}
