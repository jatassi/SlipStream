import { Link } from "@tanstack/react-router";

import type { HealthVolume, HealthVolumeLevel, NodesNode } from "../api/gen/model";
import { levels } from "../health/levels";
import { NodeDot } from "../nodes/state-mark";
import { Free } from "../roots/free";
import { Data } from "../ui/data";
import { Empty } from "../ui/empty";
import { Meter } from "../ui/meter";
import { Section } from "../ui/section";
import { SignalChip } from "../ui/signal-chip";
// oxlint-disable-next-line import/max-dependencies -- a volume's row draws its level, meter and free space
import { bodyText } from "../ui/type";
import { seeAllLink } from "./jobs";

const rank = (level: HealthVolumeLevel) => (level === null ? 2 : levels[level].rank);

type NodeVolumes = { readonly nodeId: string; readonly name: string; volumes: HealthVolume[] };

/** Volumes by Node, the Node with the worst volume first, and in each the worst first, then the least free. */
export function storageOrder(volumes: readonly HealthVolume[]): NodeVolumes[] {
  const byNode = new Map<string, NodeVolumes>();
  for (const volume of volumes) {
    const group = byNode.get(volume.nodeId);
    if (group === undefined) {
      byNode.set(volume.nodeId, {
        nodeId: volume.nodeId,
        name: volume.nodeName,
        volumes: [volume],
      });
    } else {
      group.volumes.push(volume);
    }
  }
  const worst = ({ volumes: held }: NodeVolumes) =>
    Math.min(...held.map(({ level }) => rank(level)));
  for (const group of byNode.values()) {
    group.volumes.sort((a, b) => rank(a.level) - rank(b.level) || a.free - b.free);
  }
  return [...byNode.values()].toSorted((a, b) => worst(a) - worst(b));
}

function StorageRow({ volume }: { volume: HealthVolume }) {
  const level = volume.level ?? undefined;
  return (
    <li
      className={`flex flex-col gap-2 px-4 py-3 ${bodyText} not-first:shadow-[inset_0_1px_0_var(--color-hairline)]`}
    >
      <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
        <span className="flex min-w-0 items-center gap-2">
          {level !== undefined && <SignalChip kind={level}>{levels[level].chip}</SignalChip>}
          <Data className="truncate">{volume.name}</Data>
        </span>
        <Free {...volume} />
      </div>
      <Meter
        label={`${volume.name} used`}
        value={volume.total - volume.free}
        max={volume.total}
        tone={level}
      />
    </li>
  );
}

/** One meter per volume Health watches, grouped by Node and coloured by Health's levels. */
export function Storage({
  volumes,
  nodes,
}: {
  volumes: readonly HealthVolume[];
  nodes: readonly NodesNode[];
}) {
  return (
    <Section title="Storage">
      <div className="flex flex-col gap-3">
        {volumes.length === 0 ? (
          <Empty look="sheet">
            No Node has reported a volume yet. Each library root and downloads folder shows here
            once its Node reports its free space.
          </Empty>
        ) : (
          storageOrder(volumes).map((group) => {
            const node = nodes.find(({ id }) => id === group.nodeId);
            return (
              <div key={group.nodeId} className="flex flex-col gap-2">
                <h3 className={`flex items-center gap-2 font-semibold text-ink-2 ${bodyText}`}>
                  {node !== undefined && <NodeDot node={node} />}
                  {group.name}
                </h3>
                <ul
                  aria-label={`Volumes on ${group.name}`}
                  className="rounded-card bg-glass ring-1 ring-hairline ring-inset"
                >
                  {group.volumes.map((volume) => (
                    <StorageRow key={volume.mount} volume={volume} />
                  ))}
                </ul>
              </div>
            );
          })
        )}
        <Link
          to="/settings/$section"
          params={{ section: "library-roots" }}
          preload={false}
          className={seeAllLink}
        >
          Library roots
        </Link>
      </div>
    </Section>
  );
}
