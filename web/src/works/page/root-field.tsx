import { useListRootsSuspense, useListVolumesSuspense } from "../../api/gen/query/roots/roots";
import type { WorksContentType } from "../../api/gen/model";
import { Select } from "../../ui/select";
import { freeOf } from "../../roots/root-columns";

/** The Work's library root, fixed while it has files: a Work with any keeps its root until Moves. */
export function RootField({
  type,
  value,
  files,
  onValueChange,
}: {
  type: WorksContentType;
  value: string;
  files: number;
  onValueChange: (root: string) => void;
}) {
  const roots = useListRootsSuspense().data.roots.filter((root) => root.contentType === type.id);
  const { nodes } = useListVolumesSuspense().data;
  return (
    <div className="flex flex-col gap-1.5">
      <Select
        label="Library root"
        options={roots.map((candidate) => ({
          value: candidate.id,
          label: `${candidate.path} · ${freeOf(candidate, nodes)}`,
        }))}
        value={value}
        disabled={files > 0}
        onValueChange={onValueChange}
      />
      {files > 0 && (
        <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
          This {type.work.one.toLowerCase()} has files, so it stays in its library root. Moving a
          Work with files arrives with Moves.
        </p>
      )}
    </div>
  );
}
