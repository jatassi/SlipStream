import { isProblem, sentenceOf } from "../api/errors";
import type { RootsRoot } from "../api/gen/model";
import { markRootGone } from "../api/gen/query/roots/roots";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { toast } from "../ui/toast";
import { filesWords } from "./refused";

const counted = new Intl.NumberFormat("en");

function goneRefusal(error: unknown): string {
  return isProblem(error, "scan_not_stopped")
    ? "This root's last library scan didn't stop, so no files are waiting to be marked gone."
    : sentenceOf(error);
}

/** Confirm the gone files that stopped a root's last library scan; the scan runs again and marks them. */
export function MarkGoneDialog({
  root,
  name,
  onOpenChange,
}: {
  root: RootsRoot;
  name: string;
  onOpenChange: (open: boolean) => void;
}) {
  const { stopped } = root.scan;
  if (stopped === null) {
    return null;
  }
  const confirm = async () => {
    try {
      await markRootGone(root.id);
    } catch (error) {
      throw new Error(goneRefusal(error), { cause: error });
    }
    toast(`Library scan is marking ${filesWords(stopped.gone)} gone in the ${name}.`);
  };
  return (
    <ConfirmDialog
      title={`Mark ${filesWords(stopped.gone)} gone?`}
      confirmLabel="Mark them gone"
      open
      onOpenChange={onOpenChange}
      onConfirm={confirm}
    >
      <p>
        Library scan didn&apos;t find {counted.format(stopped.gone)} of the {filesWords(stopped.of)}{" "}
        in the {name}. Their records stay, and their cells read missing until the files come back or
        you forget them on each Work&apos;s page.
      </p>
      <p>If the disk or share isn&apos;t mounted right, fix that and scan again instead.</p>
    </ConfirmDialog>
  );
}
