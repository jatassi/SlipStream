import { useState } from "react";
import { z } from "zod";

import type { IdentityPasskey } from "../api/gen/model";
import { useDeletePasskey, useUpdatePasskey } from "../api/gen/query/identity/identity";
import { Button } from "../ui/button";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { Dialog, DialogClose } from "../ui/dialog";
import { submitErrors, useAppForm } from "../ui/form";
import { DialogActions } from "./dialog-actions";
import { nameField } from "./display-name";

export type Acting = {
  readonly kind: "rename" | "delete";
  readonly passkey: IdentityPasskey;
} | null;

const renameForm = z.object({ name: nameField });

function RenameForm({ passkey, onDone }: { passkey: IdentityPasskey; onDone: () => void }) {
  const rename = useUpdatePasskey();
  const form = useAppForm({
    defaultValues: { name: passkey.name },
    validators: { onSubmit: renameForm },
    onSubmit: async ({ value, formApi }) => {
      try {
        await rename.mutateAsync({ id: passkey.id, data: renameForm.parse(value) });
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
        <form.AppField name="name">{(field) => <field.TextField label="Name" />}</form.AppField>
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

export function PasskeyDialogs({ acting, act }: { acting: Acting; act: (acting: Acting) => void }) {
  const remove = useDeletePasskey();
  const [shown, setShown] = useState(acting);
  if (acting !== null && acting !== shown) {
    setShown(acting);
  }
  const close = (open: boolean) => {
    if (!open) {
      act(null);
    }
  };
  return (
    <>
      <Dialog title="Rename passkey" open={acting?.kind === "rename"} onOpenChange={close}>
        {shown?.kind === "rename" && (
          <RenameForm
            key={shown.passkey.id}
            passkey={shown.passkey}
            onDone={() => {
              act(null);
            }}
          />
        )}
      </Dialog>
      <ConfirmDialog
        title="Delete this passkey?"
        confirmLabel="Delete"
        look="fail"
        open={acting?.kind === "delete"}
        onOpenChange={close}
        onConfirm={() =>
          shown === null ? undefined : remove.mutateAsync({ id: shown.passkey.id })
        }
      >
        {shown && (
          <p>
            {shown.passkey.name} will no longer sign you in at {shown.passkey.host}.
          </p>
        )}
      </ConfirmDialog>
    </>
  );
}
