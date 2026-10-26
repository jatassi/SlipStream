import type { NodesNode, RootsNodeVolumes, RootsRoot, WorksContentType } from "../api/gen/model";
import { bytes, countWords } from "../format";
import { NodeDot } from "../routes/_app/nodes/-state-dot";
import { Chip } from "../ui/chip";
import { Data } from "../ui/data";
import type { Column } from "../ui/table";
import type { Acting } from "./acting-dialogs";
import { RootMenu } from "./root-menu";
import { stateWords } from "./words";

const counted = new Intl.NumberFormat("en");

function FolderCell({ root, node }: { root: RootsRoot; node: string }) {
  const state = stateWords[root.state];
  return (
    <span className="flex flex-col gap-0.5">
      <span className="flex flex-wrap items-center gap-2">
        <Data className="break-all">{root.path}</Data>
        {root.default && <Chip>Default</Chip>}
      </span>
      {root.rehome !== undefined && (
        <span className="text-meta text-ink-3">
          Checking its files at a new place, {counted.format(root.rehome.checked)} of{" "}
          {counted.format(root.rehome.total)}
        </span>
      )}
      {state.tone !== "ready" && (
        <span className={`text-meta ${state.tone === "warn" ? "text-warn-ink" : "text-ink-3"}`}>
          {state.words(node)}
        </span>
      )}
    </span>
  );
}

function NodeCell({ node }: { node: NodesNode | undefined }) {
  return node === undefined ? (
    "A removed Node"
  ) : (
    <span className="flex items-center gap-2">
      <NodeDot node={node} />
      {node.name}
    </span>
  );
}

export function freeOf(root: RootsRoot, volumes: readonly RootsNodeVolumes[]): string {
  const volume = volumes
    .find((group) => group.nodeId === root.nodeId)
    ?.volumes.find((candidate) => candidate.id === root.volumeId);
  return volume === undefined ? "Not read yet" : `${bytes(volume.free)} free`;
}

export function rootColumns(
  type: WorksContentType,
  nodes: readonly NodesNode[],
  volumes: readonly RootsNodeVolumes[],
  onAct: (acting: Acting) => void,
): readonly [Column<RootsRoot>, ...Column<RootsRoot>[]] {
  const nodeOf = (root: RootsRoot) => nodes.find((node) => node.id === root.nodeId);
  return [
    {
      id: "folder",
      header: "Folder",
      cell: (root) => <FolderCell root={root} node={nodeOf(root)?.name ?? "Its Node"} />,
    },
    { id: "node", header: "Node", cell: (root) => <NodeCell node={nodeOf(root)} /> },
    { id: "free", header: "Free", cell: (root) => freeOf(root, volumes) },
    {
      id: "works",
      header: type.work.many,
      cell: (root) => `${countWords(root.works, type.work)}, ${bytes(root.fileBytes)}`,
    },
    {
      id: "actions",
      header: "Actions",
      align: "end",
      cell: (root) => <RootMenu root={root} label={type.work.many} onAct={onAct} />,
    },
  ];
}
