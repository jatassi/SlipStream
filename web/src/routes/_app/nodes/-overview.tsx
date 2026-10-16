import { useListNodesSuspense } from "../../../api/gen/query/nodes/nodes";
import { Empty } from "../../../shell/empty";
import { useShell } from "../../../ui/shell-context";
import { Facts } from "./-facts";
import { nodeOf } from "./-node";
import { PhoneNodePicker } from "./-node-picker";
import { SectionList } from "./-section-list";

export function Overview({ nodeId }: { nodeId: string }) {
  const { nodes } = useListNodesSuspense().data;
  const node = nodeOf(nodes, nodeId);
  const phone = useShell() === "phone";
  return (
    <>
      {phone && nodes.length > 1 && <PhoneNodePicker nodes={nodes} current={node} />}
      {node.self ? (
        <Facts />
      ) : (
        <Empty>{node.name}&apos;s details aren&apos;t shown here yet.</Empty>
      )}
      {phone && <SectionList node={node} />}
    </>
  );
}
