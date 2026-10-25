import {
  Delete02Icon,
  Megaphone01Icon,
  MoreHorizontalIcon,
  PauseIcon,
  PlayIcon,
  RefreshIcon,
  UnavailableIcon,
} from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";

import { IconButton } from "../ui/icon-button";
import { Menu, type MenuEntry } from "../ui/menu";
import { type Subject, type TorrentAction, actionLabels } from "./actions";
import { useActing } from "./acting";

export const actionIcons: { readonly [A in TorrentAction]: IconSvgElement } = {
  pause: PauseIcon,
  resume: PlayIcon,
  announce: Megaphone01Icon,
  recheck: RefreshIcon,
  "mark-failed": UnavailableIcon,
  remove: Delete02Icon,
};

const parting = new Set<TorrentAction>(["remove", "mark-failed"]);

/** Menu items for actions, the ones that part with a torrent last and in red. */
export function menuItems(
  actions: readonly TorrentAction[],
  onSelect: (action: TorrentAction) => void,
): MenuEntry[] {
  const item = (action: TorrentAction): MenuEntry => ({
    label: actionLabels[action],
    icon: actionIcons[action],
    ...(parting.has(action) && { tone: "danger" as const }),
    onSelect: () => {
      onSelect(action);
    },
  });
  const staying = actions.filter((action) => !parting.has(action));
  const going = actions.filter((action) => parting.has(action));
  return [
    ...staying.map((action) => item(action)),
    ...(staying.length > 0 && going.length > 0 ? (["separator"] as const) : []),
    ...going.map((action) => item(action)),
  ];
}

const noEntries: readonly MenuEntry[] = [];

export function ActionsMenu({
  subject,
  actions,
  extra = noEntries,
}: {
  subject: Subject;
  actions: readonly TorrentAction[];
  extra?: readonly MenuEntry[];
}) {
  const { start, dialogs } = useActing();
  const items = [
    ...extra,
    ...(extra.length > 0 && actions.length > 0 ? (["separator"] as const) : []),
    ...menuItems(actions, (action) => {
      start(action, [subject]);
    }),
  ];
  if (items.length === 0) {
    return null;
  }
  const label = `${subject.name} actions`;
  return (
    <>
      <Menu
        label={label}
        trigger={<IconButton icon={MoreHorizontalIcon} label={label} />}
        items={items}
      />
      {dialogs}
    </>
  );
}
