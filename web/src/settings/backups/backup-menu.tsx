import {
  Delete02Icon,
  Download04Icon,
  MoreHorizontalIcon,
  RefreshIcon,
} from "@hugeicons/core-free-icons";

import { sentenceOf } from "../../api/errors";
import type { BackupsBackup } from "../../api/gen/model";
import { IconButton } from "../../ui/icon-button";
import { Menu } from "../../ui/menu";
import { toast } from "../../ui/toast";
import { downloadBackup } from "./downloads";

export function BackupMenu({
  backup,
  when,
  onRestore,
  onDelete,
}: {
  backup: BackupsBackup;
  when: string;
  onRestore: () => void;
  onDelete: () => void;
}) {
  return (
    <Menu
      label={`Backup from ${when}`}
      trigger={
        <IconButton icon={MoreHorizontalIcon} label={`Actions for the Backup from ${when}`} />
      }
      items={[
        {
          label: "Download",
          icon: Download04Icon,
          disabled: !backup.local || backup.state !== "verified",
          onSelect: () => {
            downloadBackup(backup.id).catch((error: unknown) => {
              toast(sentenceOf(error));
            });
          },
        },
        {
          label: "Restore…",
          icon: RefreshIcon,
          disabled: backup.state !== "verified",
          onSelect: onRestore,
        },
        "separator",
        {
          label: "Delete…",
          icon: Delete02Icon,
          tone: "danger",
          disabled: backup.state === "taking" || backup.state === "deleting",
          onSelect: onDelete,
        },
      ]}
    />
  );
}
