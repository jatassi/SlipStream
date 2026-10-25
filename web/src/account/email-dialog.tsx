import { useState } from "react";
import { z } from "zod";

import { isProblem } from "../api/errors";
import type { IdentityEmailChangeStarted } from "../api/gen/model";
import { useStartEmailChange } from "../api/gen/query/identity/identity";
import { StartEmailChangeBody } from "../api/gen/zod/identity/identity.zod";
import { Button } from "../ui/button";
import { Dialog, DialogClose } from "../ui/dialog";
import { submitErrors, useAppForm } from "../ui/form";
import { DialogActions } from "./dialog-actions";

const emailForm = z.object({ email: z.string().trim().pipe(StartEmailChangeBody.shape.email) });

const addressUnset = "Your Admin hasn't set SlipStream's address yet, so email changes don't work.";

function sentence(email: string, { email: { outcome, detail } }: IdentityEmailChangeStarted) {
  switch (outcome) {
    case "sent": {
      return `We sent a confirm link to ${email}. Your email changes once you follow it.`;
    }
    case "failed": {
      return `The confirm link to ${email} didn't send. The mail server responded: ${detail ?? "nothing"}`;
    }
    case "not-set-up": {
      return "SlipStream can't send email yet. Ask your Admin to change your email for you.";
    }
    default: {
      return outcome satisfies never;
    }
  }
}

function EmailForm({ onSent }: { onSent: (outcome: string) => void }) {
  const start = useStartEmailChange();
  const form = useAppForm({
    defaultValues: { email: "" },
    validators: { onSubmit: emailForm },
    onSubmit: async ({ value, formApi }) => {
      const { email } = emailForm.parse(value);
      try {
        onSent(sentence(email, await start.mutateAsync({ data: { email } })));
      } catch (error) {
        formApi.setErrorMap({
          onSubmit: isProblem(error, "address_unset")
            ? { form: addressUnset, fields: {} }
            : submitErrors(error, { email_taken: "email" }),
        });
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
        SlipStream sends a confirm link to the new address, and your email changes once you follow
        it.
      </p>
      <form.AppForm>
        <form.AppField name="email">
          {(field) => <field.TextField label="New email" type="email" autoComplete="email" />}
        </form.AppField>
        <form.FormAlert />
        <DialogActions>
          <DialogClose>
            <Button>Cancel</Button>
          </DialogClose>
          <form.SubmitButton label="Send confirm link" />
        </DialogActions>
      </form.AppForm>
    </form>
  );
}

function EmailChange() {
  const [outcome, setOutcome] = useState<string>();
  if (outcome === undefined) {
    return <EmailForm onSent={setOutcome} />;
  }
  return (
    <>
      <output className="block text-ink-1">{outcome}</output>
      <DialogActions>
        <DialogClose>
          <Button look="primary">Done</Button>
        </DialogClose>
      </DialogActions>
    </>
  );
}

export function EmailDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog title="Change email" open={open} onOpenChange={onOpenChange}>
      <EmailChange />
    </Dialog>
  );
}
