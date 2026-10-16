import { Delete02Icon, MoreHorizontalIcon, PencilEdit02Icon } from "@hugeicons/core-free-icons";

import type { IdentityPasskey } from "../api/gen/model";
import { useListPasskeysSuspense } from "../api/gen/query/identity/identity";
import { clock } from "../format";
import { IconButton } from "../ui/icon-button";
import { List, ListRow } from "../ui/list";
import { Menu } from "../ui/menu";
import type { Acting } from "./passkey-dialogs";

function metaOf({ host, createdAt, lastUsedAt }: IdentityPasskey): string {
  const used = lastUsedAt === undefined ? "Never used" : `Last used ${clock(lastUsedAt)}`;
  return `${host} · Added ${clock(createdAt)} · ${used}`;
}

export function PasskeyRows({ act }: { act: (acting: Acting) => void }) {
  const { passkeys } = useListPasskeysSuspense().data;
  if (passkeys.length === 0) {
    return <p className="text-ink-3">You have no passkeys yet.</p>;
  }
  return (
    <List label="Passkeys">
      {passkeys.map((passkey) => (
        <ListRow
          key={passkey.id}
          title={passkey.name}
          meta={metaOf(passkey)}
          trailing={
            <Menu
              label={`${passkey.name} actions`}
              trigger={<IconButton icon={MoreHorizontalIcon} label={`${passkey.name} actions`} />}
              items={[
                {
                  label: "Rename",
                  icon: PencilEdit02Icon,
                  onSelect: () => {
                    act({ kind: "rename", passkey });
                  },
                },
                {
                  label: "Delete",
                  icon: Delete02Icon,
                  tone: "danger",
                  onSelect: () => {
                    act({ kind: "delete", passkey });
                  },
                },
              ]}
            />
          }
        />
      ))}
    </List>
  );
}
