import { Button } from "../ui/button";
import { type Subject, type TorrentAction, actionLabels, menuFor } from "./actions";
import { useActing } from "./acting";
import { actionIcons } from "./actions-menu";
import type { Row } from "./model";

const bulkActions = ["pause", "resume", "remove"] as const satisfies readonly TorrentAction[];

/** What a picked set of torrents can do together: pause, resume and remove those that offer it. */
export function BulkBar({
  picked,
  subjectOf,
  onClear,
}: {
  picked: readonly Row[];
  subjectOf: (row: Row) => Subject;
  onClear: () => void;
}) {
  const { start, dialogs } = useActing();
  const offering = (action: TorrentAction) =>
    picked.filter((row) => menuFor(row).includes(action)).map((row) => subjectOf(row));
  return (
    <div
      role="toolbar"
      aria-label="Picked torrents"
      className="mb-4 flex flex-wrap items-center gap-2 rounded-card bg-glass px-4 py-2 ring-1 ring-hairline ring-inset"
    >
      <span className="mr-auto text-ink-1">
        {picked.length} {picked.length === 1 ? "torrent" : "torrents"} picked
      </span>
      {bulkActions.map((action) => {
        const subjects = offering(action);
        return (
          subjects.length > 0 && (
            <Button
              key={action}
              icon={actionIcons[action]}
              look={action === "remove" ? "ghost" : "default"}
              onClick={() => {
                start(action, subjects);
              }}
            >
              {subjects.length === picked.length
                ? actionLabels[action]
                : `${actionLabels[action]} ${subjects.length}`}
            </Button>
          )
        );
      })}
      <Button look="ghost" onClick={onClear}>
        Clear
      </Button>
      {dialogs}
    </div>
  );
}
