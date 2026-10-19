import type { NodesNode, RootsNodeVolumes, RootsRoot, RootsRootVolume } from "../api/gen/model";
import { bytes, clock } from "../format";
import { NodeDot } from "../routes/_app/nodes/-state-dot";
import { Chip } from "../ui/chip";
import { Data } from "../ui/data";
import { Meter } from "../ui/meter";
import { Section } from "../ui/section";

const kindChip = { local: undefined, removable: "Removable", network: "Network" } as const;

function VolumeRow({
  volume,
  live,
  roots,
}: {
  volume: RootsRootVolume;
  live: boolean;
  roots: readonly RootsRoot[];
}) {
  const held = roots.filter((root) => volume.rootIds.includes(root.id));
  const chip = kindChip[volume.kind];
  const free = `${bytes(volume.free)} free of ${bytes(volume.total)}`;
  return (
    <li className="flex flex-col gap-2 px-4 py-3.5 not-first:shadow-[inset_0_1px_0_var(--color-hairline)]">
      <div className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
        <span className="flex min-w-0 items-center gap-2">
          <Data className="truncate">{volume.name}</Data>
          {chip !== undefined && <Chip>{chip}</Chip>}
        </span>
        <span className="text-ink-2 tabular-nums">
          {live ? free : `${free} as of ${clock(volume.sampledAt)}`}
        </span>
      </div>
      <Meter label={`${volume.name} used`} value={volume.total - volume.free} max={volume.total} />
      {held.length > 0 && (
        <p className="flex flex-wrap gap-x-3 text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
          {held.map((root) => (
            <Data key={root.id} className="break-all">
              {root.path}
            </Data>
          ))}
        </p>
      )}
    </li>
  );
}

/** Each volume that holds a root, grouped by its Node: how full it is and the roots on it. */
export function StorageBlock({
  volumes,
  roots,
  nodes,
}: {
  volumes: readonly RootsNodeVolumes[];
  roots: readonly RootsRoot[];
  nodes: readonly NodesNode[];
}) {
  return (
    <Section title="Storage">
      {volumes.length === 0 ? (
        <p className="text-ink-3">No Storage Node has reported a volume yet.</p>
      ) : (
        <div className="flex flex-col gap-4">
          {volumes.map((group) => {
            const node = nodes.find((candidate) => candidate.id === group.nodeId);
            return (
              <div key={group.nodeId} className="flex flex-col gap-2">
                <h3 className="flex items-center gap-2 font-semibold text-ink-2">
                  {node !== undefined && <NodeDot node={node} />}
                  {node?.name ?? "A removed Node"}
                </h3>
                <ul
                  aria-label={`Volumes on ${node?.name ?? "a removed Node"}`}
                  className="rounded-card bg-glass ring-1 ring-hairline ring-inset"
                >
                  {group.volumes.map((volume) => (
                    <VolumeRow key={volume.id} volume={volume} live={group.live} roots={roots} />
                  ))}
                </ul>
              </div>
            );
          })}
        </div>
      )}
    </Section>
  );
}
