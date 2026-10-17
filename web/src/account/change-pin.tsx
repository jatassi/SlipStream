import { useState } from "react";

import { isProblem } from "../api/errors";
import { useChangePIN } from "../api/gen/query/identity/identity";
import { pinErrors } from "../session/pin-errors";
import { Button } from "../ui/button";
import { Dialog, DialogClose } from "../ui/dialog";
import { useAppForm } from "../ui/form";
import { pinLength } from "../ui/pin-field";
import { toast } from "../ui/toast";
import { DialogActions } from "./dialog-actions";

type Stage = "current" | "new";

// The current PIN comes first and the new one after it, so a touch screen
// shows one keypad at a time. The Node checks both together.
function useChangePinForm(onDone: () => void) {
  const change = useChangePIN();
  const [stage, setStage] = useState<Stage>("current");
  const form = useAppForm({
    defaultValues: { current: "", new: "" },
    onSubmit: async ({ value, formApi }) => {
      if (stage === "current") {
        if (value.current.length === pinLength) {
          setStage("new");
        }
        return;
      }
      try {
        await change.mutateAsync({ data: value });
        toast("Your PIN is changed.");
        onDone();
      } catch (error) {
        formApi.reset();
        setStage("current");
        formApi.setErrorMap({
          onSubmit: isProblem(error, "pin_wrong")
            ? { form: "That PIN is wrong.", fields: {} }
            : pinErrors(error),
        });
      }
    },
  });
  return { form, stage };
}

function ChangePinForm({ onDone }: { onDone: () => void }) {
  const { form, stage } = useChangePinForm(onDone);
  return (
    <form
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <form.AppForm>
        {stage === "current" ? (
          <form.AppField name="current">
            {(field) => (
              <field.PinPad label="Current PIN" autoComplete="current-password" autoSubmit />
            )}
          </form.AppField>
        ) : (
          <form.AppField name="new">{(field) => <field.NewPinPad />}</form.AppField>
        )}
        <form.FormAlert />
        <DialogActions>
          <DialogClose>
            <Button>Cancel</Button>
          </DialogClose>
        </DialogActions>
      </form.AppForm>
    </form>
  );
}

export function ChangePinDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog title="Change PIN" open={open} onOpenChange={onOpenChange}>
      <ChangePinForm
        onDone={() => {
          onOpenChange(false);
        }}
      />
    </Dialog>
  );
}
