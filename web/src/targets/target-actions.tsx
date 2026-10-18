import { Copy01Icon, Delete02Icon, MoreHorizontalIcon } from "@hugeicons/core-free-icons";

import { isProblem, sentenceOf } from "../api/errors";
import type { TargetsTarget } from "../api/gen/model";
import { deleteTarget, duplicateTarget } from "../api/gen/query/targets/targets";
import { IconButton } from "../ui/icon-button";
import { Menu } from "../ui/menu";
import { toast } from "../ui/toast";

export type Acting = { readonly kind: "delete"; readonly target: TargetsTarget };

async function duplicating(target: TargetsTarget) {
  try {
    const copy = await duplicateTarget(target.id);
    toast(`Made ${copy.name} from ${target.name}.`);
  } catch (error) {
    toast(sentenceOf(error), { kind: "fail" });
  }
}

export function TargetMenu({
  target,
  onAct,
}: {
  target: TargetsTarget;
  onAct: (acting: Acting) => void;
}) {
  return (
    <Menu
      label={target.name}
      trigger={<IconButton icon={MoreHorizontalIcon} label={`Actions for ${target.name}`} />}
      items={[
        {
          label: "Duplicate",
          icon: Copy01Icon,
          onSelect: () => {
            void duplicating(target);
          },
        },
        "separator",
        {
          label: "Delete",
          icon: Delete02Icon,
          tone: "danger",
          onSelect: () => {
            onAct({ kind: "delete", target });
          },
        },
      ]}
    />
  );
}

export async function deleting(target: TargetsTarget) {
  try {
    await deleteTarget(target.id);
  } catch (error) {
    if (isProblem(error, "target_in_use")) {
      throw new Error(`${target.name} can't be deleted. ${error.detail ?? error.title}.`, {
        cause: error,
      });
    }
    throw error;
  }
  toast(`Deleted ${target.name}.`);
}
