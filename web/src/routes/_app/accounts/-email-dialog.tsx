import { z } from "zod";

import type { IdentityAccountRow } from "../../../api/gen/model";
import { updateAccount } from "../../../api/gen/query/identity/identity";
import { Button } from "../../../ui/button";
import { Dialog, DialogClose } from "../../../ui/dialog";
import { submitErrors, useAppForm } from "../../../ui/form";
import { toast } from "../../../ui/toast";

const emailForm = z.object({ email: z.string().trim().pipe(z.email("Enter an email address")) });

function EmailForm({ account, onDone }: { account: IdentityAccountRow; onDone: () => void }) {
  const form = useAppForm({
    defaultValues: { email: account.email },
    validators: { onSubmit: emailForm },
    onSubmit: async ({ value, formApi }) => {
      const changed = await updateAccount(account.id, emailForm.parse(value)).catch(
        (error: unknown) => {
          formApi.setErrorMap({ onSubmit: submitErrors(error, { email_taken: "email" }) });
          return null;
        },
      );
      if (changed !== null) {
        toast(`Changed ${changed.displayName}'s email to ${changed.email}`);
        onDone();
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
      <p className="text-ink-2">
        It changes at once, with no confirm link. SlipStream emails the old address about the
        change.
      </p>
      <form.AppForm>
        <form.AppField name="email">
          {(field) => <field.TextField label="Email" type="email" autoComplete="off" />}
        </form.AppField>
        <form.FormAlert />
        <div className="flex justify-end gap-2">
          <DialogClose>
            <Button>Cancel</Button>
          </DialogClose>
          <form.SubmitButton label="Change email" />
        </div>
      </form.AppForm>
    </form>
  );
}

export function EmailDialog({
  account,
  open,
  onClose,
}: {
  account: IdentityAccountRow;
  open: boolean;
  onClose: () => void;
}) {
  return (
    <Dialog
      title={`Change ${account.displayName}'s email`}
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          onClose();
        }
      }}
    >
      <EmailForm account={account} onDone={onClose} />
    </Dialog>
  );
}
