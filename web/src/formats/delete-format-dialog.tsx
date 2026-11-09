import type { FormatsCustomFormat } from "../api/gen/model";
import { deleteCustomFormat } from "../api/gen/query/formats/formats";
import { AttemptActions, useAttempt } from "../targets/attempt";
import { useImpactSheet } from "../targets/impact-sheet";
import { Dialog } from "../ui/dialog";
import { toast } from "../ui/toast";

function scoredWords({ scoredBy }: FormatsCustomFormat): string {
  const names = scoredBy.map((target) => target.name);
  const last = names.pop();
  if (last === undefined) {
    return "No Target scores it, so deleting it changes no Work.";
  }
  return names.length === 0
    ? `${last} scores it. Deleting it also deletes that score.`
    : `${names.join(", ")} and ${last} score it. Deleting it also deletes those scores.`;
}

export function DeleteFormatDialog({
  format,
  open,
  onOpenChange,
}: {
  format: FormatsCustomFormat;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const impact = useImpactSheet();
  const close = () => {
    onOpenChange(false);
  };
  const deleting = useAttempt(() =>
    impact.save({
      title: `Delete ${format.name}`,
      action: `Delete ${format.name}`,
      run: (changing) => deleteCustomFormat(format.id, changing),
      done: () => {
        toast(`Deleted ${format.name}.`);
        close();
      },
    }),
  );
  return (
    <>
      <Dialog title={`Delete ${format.name}?`} open={open} onOpenChange={onOpenChange}>
        <div className="flex flex-col gap-4">
          <p className="text-ink-2">{scoredWords(format)}</p>
          <AttemptActions attempt={deleting} look="fail" onCancel={close}>
            Delete {format.name}
          </AttemptActions>
        </div>
      </Dialog>
      {impact.sheet}
    </>
  );
}
