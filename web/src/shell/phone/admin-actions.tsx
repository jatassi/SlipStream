import { MoreHorizontalIcon, Search01Icon } from "@hugeicons/core-free-icons";

import { useAccount } from "../../session/session";
import { IconButton } from "../../ui/icon-button";
import { Avatar } from "../account";
import { openPalette } from "../palette-state";
import { openMore } from "./more-state";
import type { TabAction } from "./tab-bar";

export const adminTabActions: readonly TabAction[] = [
  { label: "Search", icon: Search01Icon, onPress: openPalette },
  { label: "More", icon: MoreHorizontalIcon, onPress: openMore },
];

export function AdminBarActions() {
  const { displayName } = useAccount();
  return (
    <>
      <IconButton icon={Search01Icon} label="Search" onClick={openPalette} />
      <button
        type="button"
        aria-label="Account"
        onClick={openMore}
        className="grid size-11 place-items-center rounded-capsule"
      >
        <Avatar name={displayName} />
      </button>
    </>
  );
}
