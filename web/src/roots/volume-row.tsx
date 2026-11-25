import type { HealthVolume, HealthVolumeLevel, RootsRoot, RootsRootVolume } from "../api/gen/model";
import { levels } from "../health/levels";
import { Chip } from "../ui/chip";
import { Data } from "../ui/data";
import { Meter } from "../ui/meter";
import { SignalChip } from "../ui/signal-chip";
import { Free } from "./free";
import { hasOwnLevels } from "./volume-levels-dialog";
import { VolumeMenu } from "./volume-menu";

const kindChip = { local: undefined, removable: "Removable", network: "Network" } as const;

/** A volume's Storage level as a chip, nothing while it has none. */
export const LevelChip = ({ level }: { level: HealthVolumeLevel | undefined }) =>
  level === null || level === undefined ? null : (
    <SignalChip kind={level}>{levels[level].chip}</SignalChip>
  );

export function VolumeRow({
  volume,
  health,
  live,
  roots,
}: {
  volume: RootsRootVolume;
  health: HealthVolume | undefined;
  live: boolean;
  roots: readonly RootsRoot[];
}) {
  const held = roots.filter((root) => volume.rootIds.includes(root.id));
  const chip = kindChip[volume.kind];
  const level = health?.level ?? undefined;
  return (
    <li className="flex flex-col gap-2 px-4 py-3.5 not-first:shadow-[inset_0_1px_0_var(--color-hairline)]">
      <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
        <span className="flex min-w-0 items-center gap-2">
          <LevelChip level={level} />
          <Data className="truncate">{volume.name}</Data>
          {chip !== undefined && <Chip>{chip}</Chip>}
          {health !== undefined && hasOwnLevels(health) && <Chip>Own levels</Chip>}
        </span>
        <span className="flex items-center gap-2">
          <Free {...volume} live={live} />
          {health !== undefined && <VolumeMenu volume={health} />}
        </span>
      </div>
      <Meter
        label={`${volume.name} used`}
        value={volume.total - volume.free}
        max={volume.total}
        tone={level}
      />
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
