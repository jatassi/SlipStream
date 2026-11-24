import { useListRootsSuspense, useListVolumesSuspense } from "../../api/gen/query/roots/roots";
import type { WorksContentType } from "../../api/gen/model";
import { Button } from "../../ui/button";
import { Select } from "../../ui/select";
import { freeOf } from "../../roots/root-columns";

/** The Work's library root: a Work with files changes root only by a Move, which "Move to…" opens. */
export function RootField({
  type,
  value,
  files,
  onValueChange,
  onMove,
}: {
  type: WorksContentType;
  value: string;
  files: number;
  onValueChange: (root: string) => void;
  onMove: () => void;
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
        <div>
          <Button size="small" onClick={onMove}>
            Move to…
          </Button>
        </div>
      )}
    </div>
  );
}
