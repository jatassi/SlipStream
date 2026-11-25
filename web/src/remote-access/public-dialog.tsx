import { DialogActions } from "../account/dialog-actions";
import { editExposure } from "../api/gen/query/reachability/reachability";
import { Button } from "../ui/button";
import { Dialog, DialogClose } from "../ui/dialog";
import { SubmitButton } from "../ui/form";
import { PublicForm } from "./public-form";

export function PublicDialog({
  id,
  subject,
  current,
  open,
  onOpenChange,
  onSaved,
}: {
  id: string;
  subject: string;
  current: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved: () => Promise<void>;
}) {
  return (
    <Dialog title={`Public address of ${subject}`} open={open} onOpenChange={onOpenChange}>
      <PublicForm
        initial={current}
        save={async (address) => {
          await editExposure(id, { public: address });
          onOpenChange(false);
          await onSaved();
        }}
        actions={
          <DialogActions>
            <DialogClose>
              <Button>Cancel</Button>
            </DialogClose>
            <SubmitButton label="Save" />
          </DialogActions>
        }
      />
    </Dialog>
  );
}
