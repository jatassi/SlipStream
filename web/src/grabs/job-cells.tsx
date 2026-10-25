import { ArrowUpRight01Icon } from "@hugeicons/core-free-icons";
import { useNavigate } from "@tanstack/react-router";

import type { GrabsGrab } from "../api/gen/model";
import { useListTorrentsSuspense } from "../api/gen/query/downloadclients/downloadclients";
import { useListTargetsSuspense } from "../api/gen/query/targets/targets";
import { useQualityNames } from "../targets/quality-names";
import { ActionsMenu } from "../torrents/actions-menu";
import { type TorrentAction, menuFor } from "../torrents/actions";
import { Chip } from "../ui/chip";

type Job = { readonly grab: GrabsGrab; readonly work: string };

export function ForCell({ grab }: Job) {
  const targets = new Map(
    useListTargetsSuspense().data.targets.map((target) => [target.id, target.name]),
  );
  const quality = useQualityNames();
  return (
    <span className="flex flex-wrap gap-1">
      {grab.quality !== undefined && <Chip look="outline">{quality(grab.quality)}</Chip>}
      {grab.targetIds.map((id) => (
        <Chip key={id}>{targets.get(id) ?? "A deleted Target"}</Chip>
      ))}
    </span>
  );
}

const jobActions = new Set<TorrentAction>(["pause", "resume", "remove"]);

/** A Grab's Torrent: open it, and while it downloads, pause, resume or remove it. */
export function JobMenu({ grab, work }: Job) {
  const navigate = useNavigate();
  const row = useListTorrentsSuspense(undefined, {
    query: { select: ({ rows }) => rows.find(({ id }) => id === grab.torrentId) },
  }).data;
  if (row === undefined) {
    return null;
  }
  const actions =
    grab.stage === "downloading" ? menuFor(row).filter((action) => jobActions.has(action)) : [];
  return (
    <ActionsMenu
      subject={{ id: row.id, name: work }}
      actions={actions}
      extra={[
        {
          label: "Open its torrent",
          icon: ArrowUpRight01Icon,
          onSelect: () => {
            void navigate({
              to: "/activity/torrents/$torrentId",
              params: { torrentId: row.id },
            });
          },
        },
      ]}
    />
  );
}
