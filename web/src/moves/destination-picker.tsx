import type { RootsRoot, WorksContentType } from "../api/gen/model";
import { useListHealthVolumesSuspense } from "../api/gen/query/health/health";
import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import { useListRootsSuspense, useListVolumesSuspense } from "../api/gen/query/roots/roots";
import { freeOf } from "../roots/root-columns";
import { healthOfRoot } from "../roots/storage-block";
import { LevelChip } from "../roots/volume-row";
import { Select } from "../ui/select";

/** Roots of the Work's Content Type to move to, each with its Node, free space and Storage level. */
export function DestinationPicker({
  roots,
  value,
  onValueChange,
}: {
  roots: readonly RootsRoot[];
  value: string;
  onValueChange: (rootId: string) => void;
}) {
  return <RootPicker label="Move to" roots={roots} value={value} onValueChange={onValueChange} />;
}

/** One of roots, each with its Node, free space and Storage level. */
export function RootPicker({
  label,
  roots,
  value,
  onValueChange,
}: {
  label: string;
  roots: readonly RootsRoot[];
  value: string;
  onValueChange: (rootId: string) => void;
}) {
  const { nodes } = useListVolumesSuspense().data;
  const levelled = useListHealthVolumesSuspense().data.volumes;
  const names = new Map(useListNodesSuspense().data.nodes.map(({ id, name }) => [id, name]));
  return (
    <Select
      label={label}
      options={roots.map((root) => ({
        value: root.id,
        label: `${root.path} on ${names.get(root.nodeId) ?? root.nodeId} · ${freeOf(root, nodes)}`,
        leading: <LevelChip level={healthOfRoot(root, nodes, levelled)?.level} />,
      }))}
      value={value}
      onValueChange={onValueChange}
    />
  );
}

export const useRootsOf = (type: WorksContentType) =>
  useListRootsSuspense().data.roots.filter((root) => root.contentType === type.id);
