import { Link01Icon } from "@hugeicons/core-free-icons";
import { useState } from "react";
import { z } from "zod";

import { isProblem } from "../api/errors";
import type { IdentityIssuedSignupLink } from "../api/gen/model";
import { createSignupLink, getAddress, setAddress } from "../api/gen/query/identity/identity";
import { Button } from "../ui/button";
import { Dialog, DialogClose } from "../ui/dialog";
import { submitErrors, useAppForm } from "../ui/form";
import { IssuedLink } from "./issued-link";

const linkForm = z.object({
  email: z.string().trim().pipe(z.email("Enter an email address")),
  address: z.string().trim(),
});

function useSignupLinkForm(onIssued: (issued: IdentityIssuedSignupLink) => void) {
  const [needsAddress, setNeedsAddress] = useState(false);
  const form = useAppForm({
    defaultValues: { email: "", address: "" },
    validators: { onSubmit: linkForm },
    onSubmit: async ({ value, formApi }) => {
      const { email, address } = linkForm.parse(value);
      try {
        if (needsAddress) {
          await setAddress({ address });
          setNeedsAddress(false);
        }
        onIssued(await createSignupLink({ email }));
      } catch (error) {
        if (isProblem(error, "address_unset")) {
          const { suggestedAddress } = await getAddress();
          formApi.setFieldValue("address", suggestedAddress);
          setNeedsAddress(true);
        }
        formApi.setErrorMap({
          onSubmit: submitErrors(error, { account_exists: "email", address_ip: "address" }),
        });
      }
    },
  });
  return { form, needsAddress };
}

function LinkForm({ onIssued }: { onIssued: (issued: IdentityIssuedSignupLink) => void }) {
  const { form, needsAddress } = useSignupLinkForm(onIssued);
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
        The person opens the link to make their own Member Account with a name and a PIN.
      </p>
      <form.AppForm>
        <form.AppField name="email">
          {(field) => <field.TextField label="Email" type="email" autoComplete="off" />}
        </form.AppField>
        {needsAddress ? (
          <form.AppField name="address">
            {(field) => (
              <field.TextField
                label="SlipStream's address"
                type="url"
                autoComplete="off"
                spellCheck={false}
                description="Links use this address. It is saved for every link after this one."
              />
            )}
          </form.AppField>
        ) : null}
        <form.FormAlert />
        <div className="flex justify-end">
          <form.SubmitButton label="Make the link" />
        </div>
      </form.AppForm>
    </form>
  );
}

export function IssuedSignupLink({ issued }: { issued: IdentityIssuedSignupLink }) {
  return (
    <div className="flex flex-col gap-6">
      <IssuedLink label="Signup link" url={issued.url} to={issued.link.email} mail={issued.email} />
      <div className="flex justify-end">
        <DialogClose>
          <Button look="primary">Done</Button>
        </DialogClose>
      </div>
    </div>
  );
}

function SignupLinkBody() {
  const [issued, setIssued] = useState<IdentityIssuedSignupLink>();
  return issued === undefined ? (
    <LinkForm onIssued={setIssued} />
  ) : (
    <IssuedSignupLink issued={issued} />
  );
}

export function NewSignupLinkDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog title="New Signup link" open={open} onOpenChange={onOpenChange}>
      <SignupLinkBody />
    </Dialog>
  );
}

export function NewSignupLinkButton({ onClick }: { onClick: () => void }) {
  return (
    <Button look="primary" icon={Link01Icon} onClick={onClick}>
      New Signup link
    </Button>
  );
}
