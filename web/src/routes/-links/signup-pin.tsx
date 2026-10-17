import { useQueryClient } from "@tanstack/react-query";

import { isProblem } from "../../api/errors";
import { useRedeemSignupLink } from "../../api/gen/query/identity/identity";
import { markSignedIn } from "../../session/session";
import { submitErrors, useAppForm } from "../../ui/form";
import type { Flow } from "./signup-steps";
import type { StepProps } from "./steps";

/** Makes the Account and signs in, so the steps after it run on the new session. */
export function Pin({ flow, next }: StepProps<Flow>) {
  const queryClient = useQueryClient();
  const join = useRedeemSignupLink();
  const form = useAppForm({
    defaultValues: { pin: "" },
    onSubmit: async ({ value, formApi }) => {
      try {
        const joined = await join.mutateAsync({
          token: flow.token,
          data: { displayName: flow.displayName, pin: value.pin },
        });
        markSignedIn(queryClient, joined);
        next();
      } catch (error) {
        if (isProblem(error, "link_dead")) {
          flow.linkDied();
          return;
        }
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
      <p>Choose a four-digit PIN. You&apos;ll type it to sign in.</p>
      <form.AppForm>
        <form.AppField name="pin">{(field) => <field.NewPinPad label="PIN" />}</form.AppField>
        <form.FormAlert />
      </form.AppForm>
    </form>
  );
}
