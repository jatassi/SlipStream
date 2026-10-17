import { Delete02Icon, MoreHorizontalIcon, PencilEdit02Icon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import type { TargetsCut } from "../api/gen/model";
import { IconButton } from "../ui/icon-button";
import { Menu } from "../ui/menu";
import { DeleteCutDialog } from "./cut-delete";
import { EditCutDialog } from "./cut-dialogs";
import { useImpactSheet } from "./impact-sheet";

type Acting = "edit" | "delete" | undefined;

/** Edit and Delete, for a custom cut. The Impact of an edit opens beside its dialog. */
export function CutMenu({ cut }: { cut: TargetsCut }) {
  const [acting, setActing] = useState<Acting>();
  const impact = useImpactSheet();
  const close = (open: boolean) => {
    if (!open) {
      setActing(undefined);
    }
  };
  return (
    <>
      <Menu
        label={cut.name}
        trigger={<IconButton icon={MoreHorizontalIcon} label={`Actions for ${cut.name}`} />}
        items={[
          {
            label: "Edit",
            icon: PencilEdit02Icon,
            onSelect: () => {
              setActing("edit");
            },
          },
          "separator",
          {
            label: "Delete",
            icon: Delete02Icon,
            tone: "danger",
            onSelect: () => {
              setActing("delete");
            },
          },
        ]}
      />
      <EditCutDialog cut={cut} open={acting === "edit"} onOpenChange={close} save={impact.save} />
      {impact.sheet}
      <DeleteCutDialog cut={cut} open={acting === "delete"} onOpenChange={close} />
    </>
  );
}
