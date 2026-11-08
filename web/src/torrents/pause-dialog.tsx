import { ConfirmDialog } from "../ui/confirm-dialog";
import type { Subject } from "./actions";

const list = new Intl.ListFormat("en", { type: "conjunction" });

type Owed = { readonly start: string; readonly within: string; readonly counts: string };

function owedOf(tracker: string | undefined, many: boolean): Owed {
  if (tracker !== undefined) {
    return { start: tracker, within: tracker, counts: "counts" };
  }
  return many
    ? { start: "Their Trackers", within: "their Trackers", counts: "count" }
    : { start: "Its Tracker", within: "its Tracker", counts: "counts" };
}

function groupSentence(tracker: string | undefined, names: readonly string[], alone: boolean) {
  const many = names.length > 1;
  const { start, within, counts } = owedOf(tracker, many);
  const which = alone ? "" : ` for ${list.format(names)}`;
  return many
    ? `${start} ${counts} no seeding time${which} while they're paused. Resume them before ${within} ${counts} a hit and run.`
    : `${start} ${counts} no seeding time${which} while it's paused. Resume it before ${within} ${counts} a hit and run.`;
}

function pauseWords(subjects: readonly Subject[]): string[] {
  const groups = new Map<string | undefined, string[]>();
  for (const { name, owing } of subjects) {
    if (owing !== undefined) {
      groups.set(owing.tracker, [...(groups.get(owing.tracker) ?? []), name]);
    }
  }
  return [...groups].map(([tracker, names]) =>
    groupSentence(tracker, names, subjects.length === 1),
  );
}

export function PauseDialog({
  subjects,
  open,
  onOpenChange,
  onConfirm,
}: {
  subjects: readonly Subject[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => Promise<void>;
}) {
  const [only, ...rest] = subjects;
  return (
    <ConfirmDialog
      title={
        only !== undefined && rest.length === 0
          ? `Pause ${only.name}?`
          : `Pause ${subjects.length} torrents?`
      }
      confirmLabel="Pause"
      open={open}
      onOpenChange={onOpenChange}
      onConfirm={onConfirm}
    >
      {pauseWords(subjects).map((sentence) => (
        <p key={sentence}>{sentence}</p>
      ))}
    </ConfirmDialog>
  );
}
