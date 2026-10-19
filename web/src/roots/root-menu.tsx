import { Delete02Icon, Edit02Icon, MoreHorizontalIcon, StarIcon } from "@hugeicons/core-free-icons";

import { sentenceOf } from "../api/errors";
import type { RootsRoot } from "../api/gen/model";
import { makeDefaultRoot } from "../api/gen/query/roots/roots";
import { IconButton } from "../ui/icon-button";
import { Menu, type MenuItem } from "../ui/menu";
import { toast } from "../ui/toast";
import type { Acting } from "./acting-dialogs";

async function makingDefault(root: RootsRoot, label: string) {
  try {
    await makeDefaultRoot(root.id);
    toast(`New ${label} now go into ${root.path}.`);
  } catch (error) {
    toast(sentenceOf(error), { kind: "fail" });
  }
}

/** Make default, Edit and Delete; a root on a written-off Node leads with Re-home. */
export function RootMenu({
  root,
  label,
  onAct,
}: {
  root: RootsRoot;
  label: string;
  onAct: (acting: Acting) => void;
}) {
  const lost = root.state === "written_off";
  const edit: MenuItem = {
    label: lost ? "Re-home" : "Edit",
    icon: Edit02Icon,
    onSelect: () => {
      onAct({ kind: "edit", rootId: root.id });
    },
  };
  const makeDefault: MenuItem[] = root.default
    ? []
    : [{ label: "Make default", icon: StarIcon, onSelect: () => void makingDefault(root, label) }];
  return (
    <Menu
      label={root.path}
      trigger={<IconButton icon={MoreHorizontalIcon} label={`Actions for ${root.path}`} />}
      items={[
        ...(lost ? [edit, ...makeDefault] : [...makeDefault, edit]),
        "separator",
        {
          label: "Delete",
          icon: Delete02Icon,
          tone: "danger",
          onSelect: () => {
            onAct({ kind: "delete", rootId: root.id });
          },
        },
      ]}
    />
  );
}
