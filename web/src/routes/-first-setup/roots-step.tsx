import { useState } from "react";

import type { NodesNode, RootsRoot, WorksContentType } from "../../api/gen/model";
import { useListNodesSuspense } from "../../api/gen/query/nodes/nodes";
import { useListRootsSuspense } from "../../api/gen/query/roots/roots";
import { useListContentTypesSuspense } from "../../api/gen/query/works/works";
import { LibraryRootDialog } from "../../roots/root-dialog";
import { Button } from "../../ui/button";
import { Data } from "../../ui/data";
import { StepFooter } from "./frame";
import type { StepBodyProps } from "./screens";

const rootOf = (roots: readonly RootsRoot[], type: WorksContentType): RootsRoot | undefined =>
  roots.find((root) => root.contentType === type.id && root.default) ??
  roots.find((root) => root.contentType === type.id);

function TypeRow({
  type,
  root,
  nodes,
  onAdd,
}: {
  type: WorksContentType;
  root: RootsRoot | undefined;
  nodes: readonly NodesNode[];
  onAdd: () => void;
}) {
  return (
    <li className="flex flex-wrap items-center gap-x-3 gap-y-1 px-4 py-3">
      <div className="flex min-w-0 flex-[1_1_16ch] flex-col">
        <span className="font-semibold text-ink-1">{type.name}</span>
        {root === undefined ? (
          <span className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
            No folder yet
          </span>
        ) : (
          <span className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
            <Data className="wrap-anywhere text-ink-2">{root.path}</Data> on{" "}
            {nodes.find((node) => node.id === root.nodeId)?.name ?? root.nodeId}
          </span>
        )}
      </div>
      {root === undefined && (
        <Button size="small" onClick={onAdd}>
          Add a folder
        </Button>
      )}
    </li>
  );
}

export function RootsStep({ nav }: StepBodyProps) {
  const { contentTypes } = useListContentTypesSuspense().data;
  const { roots } = useListRootsSuspense().data;
  const { nodes } = useListNodesSuspense().data;
  const [adding, setAdding] = useState<string>();
  const enabled = contentTypes.filter((type) => type.enabled);
  const waiting = enabled.some((type) => rootOf(roots, type) === undefined);
  return (
    <>
      <ul
        aria-label="Library roots"
        className="divide-y divide-hairline rounded-card bg-glass ring-1 ring-hairline ring-inset"
      >
        {enabled.map((type) => (
          <TypeRow
            key={type.id}
            type={type}
            root={rootOf(roots, type)}
            nodes={nodes}
            onAdd={() => {
              setAdding(type.id);
            }}
          />
        ))}
      </ul>
      {waiting && <p>Works you add wait for a root and are placed when you add one.</p>}
      <StepFooter nav={nav} />
      <LibraryRootDialog
        open={adding !== undefined}
        onOpenChange={(open) => {
          if (!open) {
            setAdding(undefined);
          }
        }}
        contentType={adding}
      />
    </>
  );
}
