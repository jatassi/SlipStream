import type {
  HealthVolumeList,
  NodesNode,
  RootsNodeVolumes,
  RootsRoot,
  RootsRootVolume,
} from "../api/gen/model";
import { useListHealthVolumesSuspense } from "../api/gen/query/health/health";
import { NodeDot } from "../routes/_app/nodes/-state-dot";
import { Section } from "../ui/section";
import { NotReporting } from "./not-reporting";
import { VolumeRow } from "./volume-row";

const healthOf = (levelled: HealthVolumeList, nodeId: string, { mount }: RootsRootVolume) =>
  levelled.volumes.find((volume) => volume.nodeId === nodeId && volume.mount === mount);

/**
 * Each volume that holds a root, grouped by its Node: how full it is, its
 * Health level and the roots on it, then the volumes whose own levels no
 * Node reports.
 */
export function StorageBlock({
  volumes,
  roots,
  nodes,
}: {
  volumes: readonly RootsNodeVolumes[];
  roots: readonly RootsRoot[];
  nodes: readonly NodesNode[];
}) {
  const levelled = useListHealthVolumesSuspense().data;
  return (
    <Section title="Storage">
      <div className="flex flex-col gap-4">
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
                      <VolumeRow
                        key={volume.id}
                        volume={volume}
                        health={healthOf(levelled, group.nodeId, volume)}
                        live={group.live}
                        roots={roots}
                      />
                    ))}
                  </ul>
                </div>
              );
            })}
          </div>
        )}
        <NotReporting volumes={levelled.notReporting} />
      </div>
    </Section>
  );
}
