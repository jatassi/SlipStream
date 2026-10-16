import { useQueryClient } from "@tanstack/react-query";
import { z } from "zod";

import { useUpdateOwnAccount } from "../api/gen/query/identity/identity";
import { markSignedIn } from "../session/session";
import { Button } from "../ui/button";
import { Dialog, DialogClose } from "../ui/dialog";
import { submitErrors, useAppForm } from "../ui/form";
import { DialogActions } from "./dialog-actions";
import { nameField } from "./display-name";

const nameForm = z.object({ displayName: nameField });

function NameForm({ current, onDone }: { current: string; onDone: () => void }) {
  const queryClient = useQueryClient();
  const update = useUpdateOwnAccount();
  const form = useAppForm({
    defaultValues: { displayName: current },
    validators: { onSubmit: nameForm },
    onSubmit: async ({ value, formApi }) => {
      try {
        markSignedIn(queryClient, await update.mutateAsync({ data: nameForm.parse(value) }));
        onDone();
      } catch (error) {
        formApi.setErrorMap({ onSubmit: submitErrors(error) });
      }
    },
  });
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
        <form.AppField name="displayName">
          {(field) => <field.TextField label="Display name" autoComplete="name" />}
        </form.AppField>
        <form.FormAlert />
        <DialogActions>
          <DialogClose>
            <Button>Cancel</Button>
          </DialogClose>
          <form.SubmitButton label="Save" />
        </DialogActions>
      </form.AppForm>
    </form>
  );
}

export function NameDialog({
  current,
  open,
  onOpenChange,
}: {
  current: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog title="Change display name" open={open} onOpenChange={onOpenChange}>
      <NameForm
        current={current}
        onDone={() => {
          onOpenChange(false);
        }}
      />
    </Dialog>
  );
}
