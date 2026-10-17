import { Add01Icon } from "@hugeicons/core-free-icons";
import { Link } from "@tanstack/react-router";
import { useState } from "react";

import type { TargetsTarget } from "../api/gen/model";
import { useListQualitiesSuspense, useListTargetsSuspense } from "../api/gen/query/targets/targets";
import { Button } from "../ui/button";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { type Column, Table } from "../ui/table";
import { useQualityNames } from "./quality-names";
import { type Acting, TargetMenu, deleting } from "./target-actions";

type Name = (key: string) => string;

const keysOf = (target: TargetsTarget): string[][] => target.tiers.map((tier) => tier ?? []);

/**
 * From the lowest quality the Target accepts to the highest, such as
 * "HDTV-720p to Remux-1080p". Within a tier, `table` (low to high) picks.
 */
export function spanOf(target: TargetsTarget, name: Name, table: readonly string[]): string {
  const tiers = keysOf(target);
  const byTable = (tier: readonly string[]) =>
    tier.toSorted((a, b) => table.indexOf(a) - table.indexOf(b));
  const best = byTable(tiers[0] ?? []).at(-1);
  const worst = byTable(tiers.at(-1) ?? [])[0];
  if (best === undefined || worst === undefined) {
    return "None";
  }
  return best === worst ? name(best) : `${name(worst)} to ${name(best)}`;
}

export function cutoffOf(target: TargetsTarget, name: Name): string {
  const tiers = keysOf(target);
  const bottom = tiers.at(-1) ?? [];
  return tiers.length > 1 && bottom.includes(target.cutoff) ? "No upgrades" : name(target.cutoff);
}

const profilesOf = (target: TargetsTarget): string =>
  target.profiles.length === 0 ? "None" : target.profiles.map((profile) => profile.name).join(", ");

function columnsOf(
  name: Name,
  table: readonly string[],
  onAct: (acting: Acting) => void,
): readonly [Column<TargetsTarget>, ...Column<TargetsTarget>[]] {
  return [
    {
      id: "name",
      header: "Name",
      cell: (target) => (
        <Link
          to="/settings/targets/$targetId"
          params={{ targetId: target.id }}
          className="font-medium hover:underline"
        >
          {target.name}
        </Link>
      ),
    },
    { id: "span", header: "Qualities", cell: (target) => spanOf(target, name, table) },
    { id: "cutoff", header: "Cutoff", cell: (target) => cutoffOf(target, name) },
    { id: "profiles", header: "Profiles", cell: profilesOf },
    {
      id: "actions",
      header: "Actions",
      align: "end",
      cell: (target) => <TargetMenu target={target} onAct={onAct} />,
    },
  ];
}

export function TargetsSection() {
  const { targets } = useListTargetsSuspense().data;
  const { qualities } = useListQualitiesSuspense().data;
  const name = useQualityNames();
  const [acting, setActing] = useState<Acting>();
  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-ink-2">
          A Target is one version of a Work to keep, such as 1080p or 2160p HDR. Profiles choose
          which Targets each Work wants.
        </p>
        <Button nativeButton={false} icon={Add01Icon} render={<Link to="/settings/targets/new" />}>
          New Target
        </Button>
      </div>
      {targets.length === 0 ? (
        <p className="text-ink-3">No Targets yet. Make one to say what a file should be.</p>
      ) : (
        <Table
          label="Targets"
          columns={columnsOf(
            name,
            qualities.map((quality) => quality.key),
            setActing,
          )}
          rows={targets}
          rowKey={(target) => target.id}
          stackBelow={560}
        />
      )}
      <ConfirmDialog
        title={`Delete ${acting?.target.name ?? "the Target"}?`}
        confirmLabel="Delete"
        look="fail"
        open={acting !== undefined}
        onOpenChange={(open) => {
          if (!open) {
            setActing(undefined);
          }
        }}
        onConfirm={() => (acting === undefined ? undefined : deleting(acting.target))}
      >
        <p>Its rules go with it. A Target that a Profile holds can&apos;t be deleted.</p>
      </ConfirmDialog>
    </div>
  );
}
