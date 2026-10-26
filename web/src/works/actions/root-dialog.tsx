import { useState } from "react";

import type { RootsNodeVolumes, RootsRoot, WorksContentType } from "../../api/gen/model";
import { useListRootsSuspense, useListVolumesSuspense } from "../../api/gen/query/roots/roots";
import { bytes } from "../../format";
import { ConfirmDialog } from "../../ui/confirm-dialog";
import { Select } from "../../ui/select";
import { act } from "./act";
import type { Subject } from "./actions";
import { filesStay, namedOf } from "./outcomes";

/** A Work for the root picker: its files decide whether it can move. */
export type Rooted = Subject & { readonly files: number };

function freeOf(root: RootsRoot, volumes: readonly RootsNodeVolumes[]): string {
  const volume = volumes
    .find((group) => group.nodeId === root.nodeId)
    ?.volumes.find((candidate) => candidate.id === root.volumeId);
  return volume === undefined ? "free space not read yet" : `${bytes(volume.free)} free`;
}

type RootDialogProps = {
  readonly type: WorksContentType;
  readonly works: readonly Rooted[];
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
};

/** Picks a library root, with each one's free space, for the Works that have no files yet. */
export function RootDialog({ type, works, open, onOpenChange }: RootDialogProps) {
  const roots = useListRootsSuspense().data.roots.filter((root) => root.contentType === type.id);
  const { nodes } = useListVolumesSuspense().data;
  const [rootId, setRootId] = useState((roots.find((root) => root.default) ?? roots[0])?.id ?? "");
  const moving = works.filter(({ files }) => files === 0);
  const staying = works.filter(({ files }) => files > 0);
  return (
    <ConfirmDialog
      title={`Set the library root of ${namedOf(works).words}`}
      confirmLabel="Set library root"
      confirmDisabled={moving.length === 0 || rootId === ""}
      open={open}
      onOpenChange={onOpenChange}
      onConfirm={() => act(works, { kind: "root", rootId })}
    >
      <Select
        label="Library root"
        options={roots.map((root) => ({
          value: root.id,
          label: `${root.path}, ${freeOf(root, nodes)}`,
        }))}
        value={rootId}
        onValueChange={setRootId}
      />
      {staying.length > 0 && <p>{filesStay(namedOf(staying))}</p>}
    </ConfirmDialog>
  );
}
