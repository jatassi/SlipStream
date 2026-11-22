import type { NodesNode } from "../../api/gen/model";
import { useListRootsSuspense } from "../../api/gen/query/roots/roots";
import { useListContentTypesSuspense } from "../../api/gen/query/works/works";
import { FolderField } from "../../folders/folder-field";
import { purposeCheck } from "../../roots/check";
import { NodeDot } from "../../nodes/state-mark";
import { Select } from "../../ui/select";

export function BackupsFolder({ node }: { node: NodesNode }) {
  const { roots } = useListRootsSuspense().data;
  const { contentTypes } = useListContentTypesSuspense().data;
  return (
    <FolderField
      label="Folder"
      nodeId={node.id}
      nodeName={node.name}
      check={purposeCheck("backups", { node: node.name, roots, types: contentTypes })}
      confirmVerb="Use"
      allowNewFolder
    />
  );
}

export function NodePicker({
  nodes,
  nodeId,
  onNodeChange,
}: {
  nodes: readonly NodesNode[];
  nodeId: string;
  onNodeChange: (nodeId: string) => void;
}) {
  return (
    <Select
      label="Node"
      options={nodes.map((each) => ({
        value: each.id,
        label: each.name,
        leading: <NodeDot node={each} />,
      }))}
      value={nodeId}
      onValueChange={onNodeChange}
    />
  );
}
