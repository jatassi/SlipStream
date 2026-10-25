import { z } from "zod";

import { nameField } from "../../account/display-name";
import type { IdentityLiveLink } from "../../api/gen/model";
import type { PasskeySupport } from "../../session/passkey";
import { Button } from "../../ui/button";
import { useAppForm } from "../../ui/form";
import { OpenSlipStream } from "./open-slipstream";
import type { StepProps } from "./steps";

export type Flow = {
  readonly token: string;
  readonly link: IdentityLiveLink;
  readonly passkeys: PasskeySupport;
  readonly displayName: string;
  readonly linkDied: () => void;
};

export function Welcome({ flow: { link }, next }: StepProps<Flow>) {
  return (
    <>
      <p className="text-ink-1">
        {link.adminName ?? "Your Admin"} invited you to {link.serverName}.
      </p>
      <p>Choose your name and a PIN to make your Account.</p>
      <Button
        look="primary"
        onClick={() => {
          next();
        }}
      >
        Get started
      </Button>
    </>
  );
}

const nameForm = z.object({ displayName: nameField });

export function Name({ flow, next }: StepProps<Flow>) {
  const form = useAppForm({
    defaultValues: { displayName: flow.displayName },
    validators: { onSubmit: nameForm },
    onSubmit: ({ value }) => {
      next(nameForm.parse(value));
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
      <p>
        You&apos;ll sign in as <span className="text-ink-1">{flow.link.email}</span>.
      </p>
      <form.AppForm>
        <form.AppField name="displayName">
          {(field) => (
            <field.TextField
              label="Your name"
              autoComplete="name"
              description="Your Admin and others see this name."
            />
          )}
        </form.AppField>
        <form.SubmitButton label="Continue" />
      </form.AppForm>
    </form>
  );
}

export function Done({ flow }: StepProps<Flow>) {
  return <OpenSlipStream sentence={`Your Account is ready, ${flow.displayName}.`} to="/portal" />;
}
