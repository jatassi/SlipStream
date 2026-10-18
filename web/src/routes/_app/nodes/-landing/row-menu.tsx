import {
  Delete02Icon,
  MoreHorizontalIcon,
  PencilEdit02Icon,
  UnavailableIcon,
} from "@hugeicons/core-free-icons";

import { IconButton } from "../../../../ui/icon-button";
import { Menu, type MenuEntry } from "../../../../ui/menu";
import type { Acting } from "../-manage/dialogs";
import type { NodeRow } from "./view";

function itemsOf(node: NodeRow, act: (acting: Acting) => void): MenuEntry[] {
  const on = (kind: Acting["kind"]) => () => {
    act({ kind, nodeId: node.id });
  };
  return [
    { label: "Rename", icon: PencilEdit02Icon, onSelect: on("rename") },
    ...(node.holds
      ? [{ label: "Write off", icon: UnavailableIcon, onSelect: on("write-off") }]
      : []),
    "separator",
    { label: "Remove", icon: Delete02Icon, tone: "danger", onSelect: on("remove") },
  ];
}

/** Rename, Write off when the Node holds work, and Remove. */
export function RowMenu({ node, act }: { node: NodeRow; act: (acting: Acting) => void }) {
  return (
    <Menu
      label={`${node.name} actions`}
      trigger={<IconButton icon={MoreHorizontalIcon} label={`${node.name} actions`} />}
      items={itemsOf(node, act)}
    />
  );
}
