import { useQueryClient } from "@tanstack/react-query";

import { isProblem } from "../../api/errors";
import type { IdentityPasskey } from "../../api/gen/model";
import { listPasskeys, useRedeemResetLink } from "../../api/gen/query/identity/identity";
import type { PasskeySupport } from "../../session/passkey";
import { markSignedIn } from "../../session/session";
import { submitErrors, useAppForm } from "../../ui/form";
import type { StepProps } from "./steps";

export type Flow = {
  readonly token: string;
  readonly email: string;
  readonly support: PasskeySupport;
  readonly passkeys: readonly IdentityPasskey[];
  readonly linkDied: () => void;
};

/** Sets the new PIN and signs in, then reads the passkeys on the new session. */
export function NewPin({ flow, next }: StepProps<Flow>) {
  const queryClient = useQueryClient();
  const redeem = useRedeemResetLink();
  const form = useAppForm({
    defaultValues: { pin: "" },
    onSubmit: async ({ value, formApi }) => {
      try {
        markSignedIn(queryClient, await redeem.mutateAsync({ token: flow.token, data: value }));
      } catch (error) {
        if (isProblem(error, "link_dead")) {
          flow.linkDied();
          return;
        }
        formApi.setErrorMap({ onSubmit: submitErrors(error) });
        return;
      }
      next({ passkeys: (await listPasskeys()).passkeys });
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
        Choose a new four-digit PIN for <span className="text-ink-1">{flow.email}</span>. Every
        device signed in as you is signed out.
      </p>
      <form.AppForm>
        <form.AppField name="pin">{(field) => <field.NewPinPad />}</form.AppField>
        <form.FormAlert />
      </form.AppForm>
    </form>
  );
}
