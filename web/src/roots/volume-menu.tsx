import { DashboardSpeed01Icon, MoreHorizontalIcon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import type { HealthVolume } from "../api/gen/model";
import { IconButton } from "../ui/icon-button";
import { Menu } from "../ui/menu";
import { VolumeLevelsDialog } from "./volume-levels-dialog";

export function VolumeMenu({ volume }: { volume: HealthVolume }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Menu
        label={volume.name}
        trigger={<IconButton icon={MoreHorizontalIcon} label={`Actions for ${volume.name}`} />}
        items={[
          {
            label: "Set warning levels",
            icon: DashboardSpeed01Icon,
            onSelect: () => {
              setOpen(true);
            },
          },
        ]}
      />
      <VolumeLevelsDialog volume={volume} open={open} onOpenChange={setOpen} />
    </>
  );
}
