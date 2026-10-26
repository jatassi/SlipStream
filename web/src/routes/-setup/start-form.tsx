import { useState } from "react";

import { Button } from "../../ui/button";
import type { FormProps } from "./form-props";
import { AlreadySetUp, Refusal } from "./refusal";
import { type StartRefusal, useStartForm } from "./use-start-form";

function StartRefused({ refused, open }: { refused: StartRefusal; open: FormProps["open"] }) {
  return refused === "already_claimed" ? (
    <AlreadySetUp />
  ) : (
    <Refusal
      text="This Node is waiting to join a SlipStream."
      action="Back to the wait"
      onAction={() => {
        open("join");
      }}
    />
  );
}

/** Start a new SlipStream: the first Admin, signed in once the Node is claimed. */
export function StartForm(props: FormProps) {
  const [refused, setRefused] = useState<StartRefusal>();
  const form = useStartForm(props, setRefused);
  if (refused !== undefined) {
    return <StartRefused refused={refused} open={props.open} />;
  }
  return (
    <form
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <p>Create the first Admin. You sign in with this email and PIN on any device.</p>
      <form.AppForm>
        <form.AppField name="email">
          {(field) => <field.TextField label="Email" type="email" autoComplete="username" />}
        </form.AppField>
        <form.AppField name="displayName">
          {(field) => <field.TextField label="Display name" autoComplete="name" />}
        </form.AppField>
        <form.AppField name="pin">
          {(field) => <field.PinField label="PIN" autoComplete="new-password" />}
        </form.AppField>
        <form.AppField name="pinAgain">
          {(field) => <field.PinField label="PIN again" autoComplete="new-password" />}
        </form.AppField>
        <form.FormAlert />
        <div className="flex flex-wrap items-center gap-3">
          <form.SubmitButton label="Create Admin" />
          <Button look="ghost" onClick={props.back}>
            Back
          </Button>
        </div>
      </form.AppForm>
    </form>
  );
}
