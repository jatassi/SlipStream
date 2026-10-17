import { Delete02Icon, MoreHorizontalIcon, StarIcon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type { TargetsProfile } from "../api/gen/model";
import { makeDefaultProfile } from "../api/gen/query/targets/targets";
import { IconButton } from "../ui/icon-button";
import { Menu } from "../ui/menu";
import { toast } from "../ui/toast";
import { DeleteProfileDialog } from "./profile-delete";

async function makingDefault(profile: TargetsProfile, works: string) {
  try {
    await makeDefaultProfile(profile.id);
    toast(`New ${works} now take ${profile.name}.`);
  } catch (error) {
    toast(sentenceOf(error), { kind: "fail" });
  }
}

/**
 * Make default and Delete, for a Profile that isn't its Content Type's
 * default. `works` names the type's Works.
 */
export function ProfileMenu({ profile, works }: { profile: TargetsProfile; works: string }) {
  const [deleting, setDeleting] = useState(false);
  if (profile.default) {
    return null;
  }
  return (
    <>
      <Menu
        label={profile.name}
        trigger={<IconButton icon={MoreHorizontalIcon} label={`Actions for ${profile.name}`} />}
        items={[
          {
            label: "Make default",
            icon: StarIcon,
            onSelect: () => {
              void makingDefault(profile, works);
            },
          },
          "separator",
          {
            label: "Delete",
            icon: Delete02Icon,
            tone: "danger",
            onSelect: () => {
              setDeleting(true);
            },
          },
        ]}
      />
      <DeleteProfileDialog profile={profile} open={deleting} onOpenChange={setDeleting} />
    </>
  );
}
