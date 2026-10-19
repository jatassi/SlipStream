import { DialogActions } from "../account/dialog-actions";
import type { RootsRoot } from "../api/gen/model";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";

/** Why the default root can't go while its Content Type has another root. */
export function KeepDefaultDialog({
  root,
  label,
  open,
  onOpenChange,
}: {
  root: RootsRoot;
  label: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog title={`Delete ${root.path}?`} open={open} onOpenChange={onOpenChange}>
      <p className="text-ink-2">
        New {label} go into this root. Make another {label} root the default first, then delete this
        one.
      </p>
      <DialogActions>
        <Button
          onClick={() => {
            onOpenChange(false);
          }}
        >
          Close
        </Button>
      </DialogActions>
    </Dialog>
  );
}
