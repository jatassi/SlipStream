import type { ImportsForceOption } from "../api/gen/model";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { nameOf } from "../works/paths";

const list = new Intl.ListFormat("en", { type: "conjunction" });

/** "This deletes Heat (1995) 1080p.mkv, because it fills no other Target." */
export function deletesWords(deletes: readonly string[]): string {
  if (deletes.length === 0) {
    return "No file is deleted. The file it replaces fills another Target, so it stays.";
  }
  const names = list.format(deletes.map((path) => nameOf(path)));
  return deletes.length === 1
    ? `This deletes ${names}, because it fills no other Target.`
    : `This deletes ${names}, because they fill no other Target.`;
}

/** Asks before a Force, naming each file it deletes. */
export function ForceConfirm({
  label,
  option,
  onForce,
  onClose,
}: {
  label: string;
  option: ImportsForceOption;
  onForce: () => Promise<unknown>;
  onClose: () => void;
}) {
  return (
    <ConfirmDialog
      title={`${label}?`}
      confirmLabel={label}
      look={option.deletes.length === 0 ? "primary" : "fail"}
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
      onConfirm={onForce}
    >
      <p>{deletesWords(option.deletes)}</p>
    </ConfirmDialog>
  );
}
