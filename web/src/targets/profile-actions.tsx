import { MoreHorizontalIcon, StarIcon } from "@hugeicons/core-free-icons";

import { sentenceOf } from "../api/errors";
import type { TargetsProfile } from "../api/gen/model";
import { makeDefaultProfile } from "../api/gen/query/targets/targets";
import { IconButton } from "../ui/icon-button";
import { Menu } from "../ui/menu";
import { toast } from "../ui/toast";

async function makingDefault(profile: TargetsProfile, works: string) {
  try {
    await makeDefaultProfile(profile.id);
    toast(`New ${works} now take ${profile.name}.`);
  } catch (error) {
    toast(sentenceOf(error), { kind: "fail" });
  }
}

/** Make default, for a Profile that isn't its Content Type's default. `works` names the type's Works. */
export function ProfileMenu({ profile, works }: { profile: TargetsProfile; works: string }) {
  if (profile.default) {
    return null;
  }
  return (
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
      ]}
    />
  );
}
