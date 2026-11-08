import {
  Delete02Icon,
  GitMergeIcon,
  MoreHorizontalIcon,
  PencilEdit02Icon,
} from "@hugeicons/core-free-icons";
import { useState } from "react";

import type { TrackersTracker } from "../api/gen/model";
import { RemoveTrackerDialog } from "../trackers/remove-tracker";
import { IconButton } from "../ui/icon-button";
import { Menu, type MenuEntry } from "../ui/menu";
import { MergeTrackerDialog } from "./merge-tracker";
import { RenameTrackerDialog } from "./rename-tracker";

type Acting = "rename" | "merge" | "remove" | undefined;

/** Rename, Merge into… while another Tracker exists, and Remove… */
export function TrackerMenu({
  tracker,
  others,
  onGone,
}: {
  tracker: TrackersTracker;
  others: readonly TrackersTracker[];
  /** Runs once the Tracker is merged away or removed, before it leaves the caches; kept is the Tracker it merged into. */
  onGone?: ((kept: TrackersTracker | undefined) => Promise<unknown>) | undefined;
}) {
  const [acting, setActing] = useState<Acting>();
  const act = (next: Acting) => () => {
    setActing(next);
  };
  const items: MenuEntry[] = [
    { label: "Rename", icon: PencilEdit02Icon, onSelect: act("rename") },
    ...(others.length === 0
      ? []
      : [{ label: "Merge into…", icon: GitMergeIcon, onSelect: act("merge") }]),
    { label: "Remove…", icon: Delete02Icon, tone: "danger", onSelect: act("remove") },
  ];
  return (
    <>
      <Menu
        label={`${tracker.name} actions`}
        trigger={<IconButton icon={MoreHorizontalIcon} label={`${tracker.name} actions`} />}
        items={items}
      />
      {acting === "rename" && <RenameTrackerDialog tracker={tracker} onClose={act(undefined)} />}
      {acting === "merge" && (
        <MergeTrackerDialog
          tracker={tracker}
          others={others}
          onClose={act(undefined)}
          onMerged={onGone}
        />
      )}
      {acting === "remove" && (
        <RemoveTrackerDialog
          tracker={tracker}
          onClose={act(undefined)}
          onRemoved={onGone && (() => onGone(undefined))}
        />
      )}
    </>
  );
}
