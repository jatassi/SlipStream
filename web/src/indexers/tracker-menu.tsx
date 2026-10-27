import { GitMergeIcon, MoreHorizontalIcon, PencilEdit02Icon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import type { IndexersTracker } from "../api/gen/model";
import { IconButton } from "../ui/icon-button";
import { Menu, type MenuEntry } from "../ui/menu";
import { MergeTrackerDialog } from "./merge-tracker";
import { RenameTrackerDialog } from "./rename-tracker";

type Acting = "rename" | "merge" | undefined;

/** Rename, and Merge into… while another Tracker exists. */
export function TrackerMenu({
  tracker,
  others,
}: {
  tracker: IndexersTracker;
  others: readonly IndexersTracker[];
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
        <MergeTrackerDialog tracker={tracker} others={others} onClose={act(undefined)} />
      )}
    </>
  );
}
