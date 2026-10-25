import { MoreHorizontalIcon } from "@hugeicons/core-free-icons";

import { Button } from "../ui/button";
import { IconButton } from "../ui/icon-button";
import { Menu } from "../ui/menu";
import { type Subject, type TorrentAction, actionLabels, menuFor } from "./actions";
import { useActing } from "./acting";
import { actionIcons, menuItems } from "./actions-menu";
import type { Row } from "./model";

const onButtons = new Set<TorrentAction>(["pause", "resume", "remove"]);

export function Actions({
  row,
  subject,
  trackersAccept,
}: {
  row: Row;
  subject: Subject;
  trackersAccept: boolean;
}) {
  const { start, dialogs } = useActing();
  const offered = menuFor(row, trackersAccept);
  const buttons = offered.filter((action) => onButtons.has(action));
  const more = offered.filter((action) => !onButtons.has(action));
  const run = (action: TorrentAction) => {
    start(action, [subject]);
  };
  if (offered.length === 0) {
    return null;
  }
  return (
    <div className="mt-3 flex flex-wrap gap-2">
      {buttons.map((action) => (
        <Button
          key={action}
          icon={actionIcons[action]}
          look={action === "remove" ? "ghost" : "default"}
          onClick={() => {
            run(action);
          }}
        >
          {actionLabels[action]}
        </Button>
      ))}
      {more.length > 0 && (
        <Menu
          label={`More actions for ${subject.name}`}
          trigger={
            <IconButton icon={MoreHorizontalIcon} label={`More actions for ${subject.name}`} />
          }
          items={menuItems(more, run)}
        />
      )}
      {dialogs}
    </div>
  );
}
